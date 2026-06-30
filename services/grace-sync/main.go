// grace-sync: fetch Grace video_steps → DataBrew batch pipeline trigger.
// Deployed as a Cloud Run Job, triggered by Cloud Scheduler.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ─── config ───────────────────────────────────────────────

type Config struct {
	Grace    GraceConfig    `json:"grace"`
	Databrew DatabrewConfig `json:"databrew"`
	Pipeline PipelineConfig `json:"pipeline"`
	Sync     SyncConfig     `json:"sync"`
}

type GraceConfig struct {
	APIURL   string `json:"api_url"`
	Username string `json:"username"`
}

type DatabrewConfig struct {
	APIURL string `json:"api_url"`
}

type PipelineConfig struct {
	TemplateID string `json:"template_id"`
	TargetID   string `json:"target_id"`
}

type SyncConfig struct {
	PageSize int    `json:"page_size"`
	StepKey  string `json:"step_key"`
}

// ─── types ────────────────────────────────────────────────

type VideoStep struct {
	VideoID string `json:"video_id"`
}

type VideoStepListResp struct {
	Data  []VideoStep `json:"data"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	Total int         `json:"total"`
}

type BatchRequest struct {
	TemplateID string   `json:"template_id"`
	AssetIDs   []string `json:"asset_ids"`
	TargetID   string   `json:"target_id"`
	Version    int      `json:"version,omitempty"`
	Name       string   `json:"name,omitempty"`
}

type BatchResponse struct {
	BatchID    string `json:"batch_id"`
	Status     string `json:"status"`
	AssetCount int    `json:"asset_count"`
	TemplateID string `json:"template_id"`
	CreatedAt  string `json:"created_at"`
}

var (
	cfg           Config
	gracePassword = os.Getenv("GRACE_PASSWORD")
	databrewToken = getEnv("DATABREW_TOKEN", "dev-token")
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	dateFlag := flag.String("date", "", "single day YYYY-MM-DD (default: yesterday)")
	fromFlag := flag.String("from", "", "date range start")
	toFlag := flag.String("to", "", "date range end (default: today)")
	idsFlag := flag.String("ids", "", "comma-separated video IDs")
	tmplFlag := flag.String("template", "", "pipeline template ID")
	targetFlag := flag.String("target", "", "execution target")
	stepKeyFlag := flag.String("step", "", "step key filter (default from config)")
	limitFlag := flag.Int("limit", 0, "cap videos (for testing)")
	configFlag := flag.String("config", "config.json", "config file path")
	flag.Parse()

	if err := loadConfig(*configFlag); err != nil {
		log.Fatalf("config: %v", err)
	}

	tmplID := cfg.Pipeline.TemplateID
	targetID := cfg.Pipeline.TargetID
	stepKey := cfg.Sync.StepKey
	if *tmplFlag != "" {
		tmplID = *tmplFlag
	}
	if *targetFlag != "" {
		targetID = *targetFlag
	}
	if *stepKeyFlag != "" {
		stepKey = *stepKeyFlag
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")
	tStart := time.Now()

	// ─── Resolve video IDs ───
	var videoIDs []string

	if *idsFlag != "" {
		for _, s := range strings.Split(*idsFlag, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				videoIDs = append(videoIDs, s)
			}
		}
		log.Printf("Using %d direct IDs", len(videoIDs))
	} else {
		var dayStart, dayEnd time.Time
		if *dateFlag != "" {
			t, err := time.ParseInLocation("2006-01-02", *dateFlag, loc)
			if err != nil {
				log.Fatalf("invalid date: %s", *dateFlag)
			}
			dayStart = t
			dayEnd = t.AddDate(0, 0, 1)
		} else if *fromFlag != "" {
			from, err := time.ParseInLocation("2006-01-02", *fromFlag, loc)
			if err != nil {
				log.Fatalf("invalid from: %s", *fromFlag)
			}
			toStr := *toFlag
			if toStr == "" {
				toStr = time.Now().In(loc).Format("2006-01-02")
			}
			to, err := time.ParseInLocation("2006-01-02", toStr, loc)
			if err != nil {
				log.Fatalf("invalid to: %s", toStr)
			}
			dayStart = from
			dayEnd = to.AddDate(0, 0, 1)
		} else {
			now := time.Now().In(loc)
			dayStart = now.Add(-1 * time.Hour)
			dayEnd = now
		}
		log.Printf("Query: %s to %s | step_key=%s", dayStart.Format("2006-01-02 15:04:05"), dayEnd.Format("2006-01-02 15:04:05"), stepKey)

		ids, err := fetchVideoSteps(dayStart, dayEnd, stepKey)
		if err != nil {
			log.Fatalf("Grace fetch: %v", err)
		}
		videoIDs = ids
	}

	if *limitFlag > 0 && *limitFlag < len(videoIDs) {
		videoIDs = videoIDs[:*limitFlag]
	}

	log.Printf("Total: %d videos | template=%s | target=%s", len(videoIDs), tmplID, targetID)

	if len(videoIDs) == 0 {
		log.Printf("No videos — exiting")
		return
	}

	// ─── Login & submit batch ───
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 30 * time.Second}

	if err := databrewLogin(client); err != nil {
		log.Fatalf("DataBrew login: %v", err)
	}
	log.Printf("DataBrew login OK")

	batchName := "grace-sync-" + time.Now().UTC().Format("20060102-150405")
	batchResp, err := submitBatch(client, tmplID, targetID, videoIDs, batchName)
	if err != nil {
		log.Fatalf("Batch submit: %v", err)
	}

	log.Printf("Batch created: %s | status=%s | %d assets in %.1fs",
		batchResp.BatchID, batchResp.Status, batchResp.AssetCount, time.Since(tStart).Seconds())
	fmt.Printf("batch_id=%s\nstatus=%s\nassets=%d\n", batchResp.BatchID, batchResp.Status, batchResp.AssetCount)
}

// ─── config loader ───────────────────────────────────────

func loadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if v := os.Getenv("GRACE_API_URL"); v != "" {
		cfg.Grace.APIURL = v
	}
	if v := os.Getenv("GRACE_USERNAME"); v != "" {
		cfg.Grace.Username = v
	}
	if v := os.Getenv("DATABREW_URL"); v != "" {
		cfg.Databrew.APIURL = v
	}
	if v := os.Getenv("PIPELINE_TEMPLATE_ID"); v != "" {
		cfg.Pipeline.TemplateID = v
	}
	if v := os.Getenv("TARGET_ID"); v != "" {
		cfg.Pipeline.TargetID = v
	}
	return nil
}

// ─── Grace API: /grace/video_steps ─────────────────────────

func fetchVideoSteps(start, end time.Time, stepKey string) ([]string, error) {
	var all []string
	page := 1

	for {
		params := url.Values{}
		params.Add("filter", fmt.Sprintf("step_key:eq:%s", stepKey))
		params.Add("filter", "status:eq:success")
		params.Add("filter", fmt.Sprintf("last_status_at:gte:%s", start.Format(time.RFC3339)))
		params.Add("filter", fmt.Sprintf("last_status_at:lt:%s", end.Format(time.RFC3339)))
		params.Set("page", strconv.Itoa(page))
		params.Set("size", strconv.Itoa(cfg.Sync.PageSize))

		reqURL := fmt.Sprintf("%s/grace/video_steps?%s", cfg.Grace.APIURL, params.Encode())
		req, _ := http.NewRequest("GET", reqURL, nil)
		req.SetBasicAuth(cfg.Grace.Username, gracePassword)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", page, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("grace API %d: %s", resp.StatusCode, string(body))
		}

		var list VideoStepListResp
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("parse page %d: %w", page, err)
		}

		for _, v := range list.Data {
			all = append(all, v.VideoID)
		}
		log.Printf("Grace page %d: %d videos (%d/%d)", page, len(list.Data), len(all), list.Total)

		if len(all) >= list.Total || len(list.Data) == 0 {
			break
		}
		page++
	}
	return all, nil
}

// ─── DataBrew API ─────────────────────────────────────────

func databrewLogin(client *http.Client) error {
	b, _ := json.Marshal(map[string]string{"token": databrewToken})
	resp, err := client.Post(cfg.Databrew.APIURL+"/auth/login", "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func submitBatch(client *http.Client, tmplID, targetID string, assetIDs []string, name string) (*BatchResponse, error) {
	b, _ := json.Marshal(BatchRequest{
		TemplateID: tmplID,
		AssetIDs:   assetIDs,
		TargetID:   targetID,
		Name:       name,
	})
	resp, err := client.Post(
		cfg.Databrew.APIURL+"/runs/batch",
		"application/json",
		bytes.NewReader(b),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("batch submit %d: %s", resp.StatusCode, string(body))
	}
	var br BatchResponse
	if err := json.Unmarshal(body, &br); err != nil {
		return nil, fmt.Errorf("parse batch response: %w", err)
	}
	return &br, nil
}

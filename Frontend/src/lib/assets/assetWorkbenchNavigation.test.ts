import { describe, it, expect, beforeEach } from "vitest";
import {
  isAssetsWorkbenchReturnUrl,
  isAssetDetailPath,
  isSafeInternalReturnUrl,
  rememberReturnUrlBeforeAssetDetail,
  consumeStoredReturnUrl,
  clearStoredAssetDetailReturn,
  ASSET_DETAIL_RETURN_SESSION_KEY,
} from "./assetWorkbenchNavigation";

describe("isAssetDetailPath", () => {
  it("detects /assets/:id", () => {
    expect(isAssetDetailPath("/assets/uuid-here")).toBe(true);
    expect(isAssetDetailPath("/assets/uuid-here?tab=algo")).toBe(true);
  });

  it("rejects workbench", () => {
    expect(isAssetDetailPath("/assets")).toBe(false);
    expect(isAssetDetailPath("/assets?preview=x")).toBe(false);
  });
});

describe("isAssetsWorkbenchReturnUrl", () => {
  it("accepts /assets with optional query", () => {
    expect(isAssetsWorkbenchReturnUrl("/assets")).toBe(true);
    expect(isAssetsWorkbenchReturnUrl("/assets?preview=abc&page=2")).toBe(true);
  });

  it("rejects asset detail paths", () => {
    expect(isAssetsWorkbenchReturnUrl("/assets/4e8496ae-c881-4860-b308-7b13c3f5d688")).toBe(
      false,
    );
  });
});

describe("isSafeInternalReturnUrl", () => {
  it("allows known app routes", () => {
    expect(isSafeInternalReturnUrl("/dashboard")).toBe(true);
    expect(isSafeInternalReturnUrl("/deliveries/abc-123")).toBe(true);
    expect(isSafeInternalReturnUrl("/assets?page=2")).toBe(true);
    expect(isSafeInternalReturnUrl("/mcap-files")).toBe(true);
  });

  it("rejects asset detail and open redirects", () => {
    expect(isSafeInternalReturnUrl("/assets/uuid")).toBe(false);
    expect(isSafeInternalReturnUrl("//evil.com")).toBe(false);
    expect(isSafeInternalReturnUrl("/unknown-app")).toBe(false);
    expect(isSafeInternalReturnUrl("/dashboard/../admin")).toBe(false);
  });
});

describe("session return URL", () => {
  beforeEach(() => {
    sessionStorage.clear();
    window.history.pushState(null, "", "/algo?page=1");
  });

  it("remember + consume roundtrip", () => {
    rememberReturnUrlBeforeAssetDetail();
    expect(sessionStorage.getItem(ASSET_DETAIL_RETURN_SESSION_KEY)).toBe("/algo?page=1");

    const url = consumeStoredReturnUrl();
    expect(url).toBe("/algo?page=1");
    expect(sessionStorage.getItem(ASSET_DETAIL_RETURN_SESSION_KEY)).toBeNull();
  });

  it("does not remember when already on asset detail", () => {
    window.history.pushState(null, "", "/assets/some-uuid");
    rememberReturnUrlBeforeAssetDetail();
    expect(sessionStorage.getItem(ASSET_DETAIL_RETURN_SESSION_KEY)).toBeNull();
  });

  it("clearStoredAssetDetailReturn removes key", () => {
    sessionStorage.setItem(ASSET_DETAIL_RETURN_SESSION_KEY, "/events");
    clearStoredAssetDetailReturn();
    expect(sessionStorage.getItem(ASSET_DETAIL_RETURN_SESSION_KEY)).toBeNull();
  });
});

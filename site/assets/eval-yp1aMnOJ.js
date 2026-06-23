import{a}from"./index-DoBiIveH.js";const c={listEvalResults:(e,s=20)=>a.get(`/assets/${e}/eval-results`,{params:{limit:s}}).then(t=>t.data),listMetrics:e=>a.get(`/assets/${e}/metrics`).then(s=>s.data),searchByMetrics:(e,s,t=1,i=20)=>a.post("/metrics:search",{filters:{lifecycle_state:s??"",metrics:e},page:t,page_size:i}).then(r=>r.data)};export{c as e};
//# sourceMappingURL=eval-yp1aMnOJ.js.map

import{r as Ae,j as se,n as Cp}from"./vendor-react-BD5WYiaW.js";import{T as Gh,I as so,R as Pp,B as Ci,c as pr,ac as Nr,ao as Dp,am as Lp,aa as Ks,X as Hu,ai as Ip,ay as Up,b4 as Np,b5 as Fp,b6 as Op,b7 as Bp,a5 as zp,b8 as kp,b9 as Vp,ba as Hh,bb as Gp,bc as Hp}from"./vendor-antd-core-n-jTm488.js";const{Text:$s}=Gh;function es(i,e){if(e>0){const s=Math.floor(e/1e6+i*1e3);return s>1e12?`${Math.floor(s/1e3)}`:`${s}`}const t=Math.floor(i/3600),n=Math.floor(i%3600/60),r=Math.floor(i%60);return t>0?`${t}:${n.toString().padStart(2,"0")}:${r.toString().padStart(2,"0")}`:`${n.toString().padStart(2,"0")}:${r.toString().padStart(2,"0")}`}function Wu(i,e){const t=i.trim();if(!t)return null;const n=t.split(":").map(Number);if(n.length===3&&n.every(s=>Number.isFinite(s)))return n[0]*3600+n[1]*60+n[2];if(n.length===2&&n.every(s=>Number.isFinite(s)))return n[0]*60+n[1];const r=parseFloat(t);return!Number.isFinite(r)||r<0?null:r>1e9&&e>0?(r*1e9-e)/1e9:r>1e12&&e>0?(r*1e6-e)/1e9:r}function Wp({onLoad:i,loading:e,range:t,onRangeChange:n,startTimestampNs:r}){const[s,a]=Ae.useState(""),[o,l]=Ae.useState(""),[c,f]=Ae.useState(""),[h,u]=Ae.useState(!1),[d,x]=Ae.useState(!1);Ae.useEffect(()=>{t&&!h&&l(es(t.startSec,r||0)),t&&!d&&f(es(t.endSec,r||0))},[t==null?void 0:t.startSec,t==null?void 0:t.endSec,h,d,r,t]);const E=()=>{const T=s.trim();T&&i(T)},m=()=>{u(!1);const T=Wu(o,r||0);if(T===null||!t)return;const P=Math.max(0,T),S=t.endSec;n==null||n({startSec:Math.min(P,S),endSec:Math.max(P,S)})},p=()=>{x(!1);const T=Wu(c,r||0);if(T===null||!t)return;const P=Math.max(0,T),S=t.startSec;n==null||n({startSec:Math.min(S,P),endSec:Math.max(S,P)})};return se.jsxs("div",{style:{display:"flex",alignItems:"center",gap:10,padding:"var(--space-3) var(--space-6)",borderBottom:"1px solid var(--gray-200)",background:"var(--gray-50)",flexWrap:"wrap"},children:[se.jsx($s,{strong:!0,style:{fontSize:"var(--font-size-xl)",whiteSpace:"nowrap",color:"var(--gray-900)"},children:"视频预览"}),se.jsx(so,{placeholder:"资产 ID",value:s,onChange:T=>a(T.target.value),onPressEnter:E,allowClear:!0,style:{maxWidth:280,fontFamily:"var(--font-mono)",fontSize:"var(--font-size-sm)"},size:"small",prefix:se.jsx(Pp,{style:{color:"var(--gray-400)"}})}),se.jsx(Ci,{type:"primary",onClick:E,loading:e,size:"small",children:"加载"}),se.jsxs("div",{style:{display:"flex",alignItems:"center",gap:2,borderLeft:"1px solid var(--gray-200)",paddingLeft:10},children:[se.jsx($s,{type:"secondary",style:{fontSize:"var(--font-size-xs)"},children:"区间"}),h?se.jsx(so,{size:"small",value:o,onChange:T=>l(T.target.value),onPressEnter:m,onBlur:m,autoFocus:!0,style:{width:180,fontFamily:"var(--font-mono)",fontSize:"var(--font-size-sm)"}}):se.jsx($s,{onClick:()=>{t?(l(es(t.startSec,r||0)),u(!0)):n==null||n({startSec:0,endSec:10})},style:{cursor:"pointer",fontFamily:"var(--font-mono)",fontSize:"var(--font-size-sm)",color:t?"var(--color-primary)":"var(--gray-400)",borderBottom:t?"1px dashed var(--color-primary)":"none",minWidth:80,textAlign:"center"},children:t?es(t.startSec,r||0):"—"}),se.jsx($s,{type:"secondary",style:{fontSize:"var(--font-size-xs)"},children:"~"}),d?se.jsx(so,{size:"small",value:c,onChange:T=>f(T.target.value),onPressEnter:p,onBlur:p,autoFocus:!0,style:{width:180,fontFamily:"var(--font-mono)",fontSize:"var(--font-size-sm)"}}):se.jsx($s,{onClick:()=>{t?(f(es(t.endSec,r||0)),x(!0)):n==null||n({startSec:0,endSec:10})},style:{cursor:"pointer",fontFamily:"var(--font-mono)",fontSize:"var(--font-size-sm)",color:t?"var(--color-primary)":"var(--gray-400)",borderBottom:t?"1px dashed var(--color-primary)":"none",minWidth:80,textAlign:"center"},children:t?es(t.endSec,r||0):"—"}),t&&se.jsx(pr,{title:"清除区间",children:se.jsx(Ci,{type:"text",size:"small",onClick:()=>n==null?void 0:n(null),style:{color:"var(--gray-400)",width:20,height:20,fontSize:12,marginLeft:2},children:"×"})})]})]})}function Xp({activeTopics:i,coverMode:e,onImport:t}){const n=Ae.useRef(null),r=Ae.useCallback(()=>{const a={version:1,exportedAt:new Date().toISOString(),activeTopics:i,coverMode:e},o=new Blob([JSON.stringify(a,null,2)],{type:"application/json"}),l=URL.createObjectURL(o),c=document.createElement("a");c.href=l,c.download=`preview-layout-${a.exportedAt.slice(0,19).replace(/[T:]/g,"-")}.json`,c.click(),URL.revokeObjectURL(l),Nr.success("布局已导出")},[i,e]),s=Ae.useCallback(a=>{var c;const o=(c=a.target.files)==null?void 0:c[0];if(!o)return;const l=new FileReader;l.onload=()=>{try{const f=JSON.parse(l.result);if(!f.version||!Array.isArray(f.activeTopics)){Nr.error("无效的布局文件");return}t({version:f.version,exportedAt:f.exportedAt||"",activeTopics:f.activeTopics,coverMode:f.coverMode??!0}),Nr.success("布局已导入")}catch{Nr.error("JSON 解析失败")}},l.readAsText(o),a.target.value=""},[t]);return se.jsxs(se.Fragment,{children:[se.jsx("input",{ref:n,type:"file",accept:".json",style:{display:"none"},onChange:s}),se.jsx(pr,{title:"导出布局",children:se.jsx(Ci,{type:"text",size:"small",icon:se.jsx(Dp,{}),onClick:r})}),se.jsx(pr,{title:"导入布局",children:se.jsx(Ci,{type:"text",size:"small",icon:se.jsx(Lp,{}),onClick:()=>{var a;return(a=n.current)==null?void 0:a.click()}})})]})}function Yp({viewport:i,sidebar:e,timeline:t}){return se.jsxs("div",{style:{display:"flex",flexDirection:"column",height:"100%",background:"var(--gray-50)"},children:[se.jsxs("div",{style:{display:"flex",flex:1,overflow:"hidden"},children:[se.jsx("div",{style:{flex:1,overflow:"hidden",display:"flex"},children:i}),se.jsx("div",{style:{width:300,flexShrink:0,borderLeft:"1px solid var(--gray-200)",background:"#fff",display:"flex",flexDirection:"column",overflow:"auto"},children:e})]}),t]})}const qp=!0,un="u-",Kp="uplot",$p=un+"hz",Zp=un+"vt",Jp=un+"title",jp=un+"wrap",Qp=un+"under",em=un+"over",tm=un+"axis",Lr=un+"off",nm=un+"select",im=un+"cursor-x",rm=un+"cursor-y",sm=un+"cursor-pt",am=un+"legend",om=un+"live",lm=un+"inline",cm=un+"series",um=un+"marker",Xu=un+"label",fm=un+"value",la="width",ca="height",Zs="top",Yu="bottom",ts="left",jo="right",Wc="#000",qu=Wc+"0",Qo="mousemove",Ku="mousedown",el="mouseup",$u="mouseenter",Zu="mouseleave",Ju="dblclick",hm="resize",dm="scroll",ju="change",go="dppxchange",Xc="--",Ns=typeof window<"u",Hl=Ns?document:null,ys=Ns?window:null,pm=Ns?navigator:null;let bt,Ra;function Wl(){let i=devicePixelRatio;bt!=i&&(bt=i,Ra&&Yl(ju,Ra,Wl),Ra=matchMedia(`(min-resolution: ${bt-.001}dppx) and (max-resolution: ${bt+.001}dppx)`),Br(ju,Ra,Wl),ys.dispatchEvent(new CustomEvent(go)))}function Zn(i,e){if(e!=null){let t=i.classList;!t.contains(e)&&t.add(e)}}function Xl(i,e){let t=i.classList;t.contains(e)&&t.remove(e)}function Vt(i,e,t){i.style[e]=t+"px"}function mi(i,e,t,n){let r=Hl.createElement(i);return e!=null&&Zn(r,e),t!=null&&t.insertBefore(r,n),r}function ri(i,e){return mi("div",i,e)}const Qu=new WeakMap;function Ti(i,e,t,n,r){let s="translate("+e+"px,"+t+"px)",a=Qu.get(i);s!=a&&(i.style.transform=s,Qu.set(i,s),e<0||t<0||e>n||t>r?Zn(i,Lr):Xl(i,Lr))}const ef=new WeakMap;function tf(i,e,t){let n=e+t,r=ef.get(i);n!=r&&(ef.set(i,n),i.style.background=e,i.style.borderColor=t)}const nf=new WeakMap;function rf(i,e,t,n){let r=e+""+t,s=nf.get(i);r!=s&&(nf.set(i,r),i.style.height=t+"px",i.style.width=e+"px",i.style.marginLeft=n?-e/2+"px":0,i.style.marginTop=n?-t/2+"px":0)}const Yc={passive:!0},mm={...Yc,capture:!0};function Br(i,e,t,n){e.addEventListener(i,t,n?mm:Yc)}function Yl(i,e,t,n){e.removeEventListener(i,t,Yc)}Ns&&Wl();function gi(i,e,t,n){let r;t=t||0,n=n||e.length-1;let s=n<=2147483647;for(;n-t>1;)r=s?t+n>>1:jn((t+n)/2),e[r]<i?t=r:n=r;return i-e[t]<=e[n]-i?t:n}function Wh(i){return(t,n,r)=>{let s=-1,a=-1;for(let o=n;o<=r;o++)if(i(t[o])){s=o;break}for(let o=r;o>=n;o--)if(i(t[o])){a=o;break}return[s,a]}}const Xh=i=>i!=null,Yh=i=>i!=null&&i>0,Co=Wh(Xh),gm=Wh(Yh);function _m(i,e,t,n=0,r=!1){let s=r?gm:Co,a=r?Yh:Xh;[e,t]=s(i,e,t);let o=i[e],l=i[e];if(e>-1)if(n==1)o=i[e],l=i[t];else if(n==-1)o=i[t],l=i[e];else for(let c=e;c<=t;c++){let f=i[c];a(f)&&(f<o?o=f:f>l&&(l=f))}return[o??Ut,l??-Ut]}function Po(i,e,t,n){let r=of(i),s=of(e);i==e&&(r==-1?(i*=t,e/=t):(i/=t,e*=t));let a=t==10?Yi:qh,o=r==1?jn:si,l=s==1?si:jn,c=o(a(on(i))),f=l(a(on(e))),h=ws(t,c),u=ws(t,f);return t==10&&(c<0&&(h=Nt(h,-c)),f<0&&(u=Nt(u,-f))),n||t==2?(i=h*r,e=u*s):(i=Jh(i,h),e=Do(e,u)),[i,e]}function qc(i,e,t,n){let r=Po(i,e,t,n);return i==0&&(r[0]=0),e==0&&(r[1]=0),r}const Kc=.1,sf={mode:3,pad:Kc},pa={pad:0,soft:null,mode:0},xm={min:pa,max:pa};function _o(i,e,t,n){return Lo(t)?af(i,e,t):(pa.pad=t,pa.soft=n?0:null,pa.mode=n?3:0,af(i,e,xm))}function vt(i,e){return i??e}function vm(i,e,t){for(e=vt(e,0),t=vt(t,i.length-1);e<=t;){if(i[e]!=null)return!0;e++}return!1}function af(i,e,t){let n=t.min,r=t.max,s=vt(n.pad,0),a=vt(r.pad,0),o=vt(n.hard,-Ut),l=vt(r.hard,Ut),c=vt(n.soft,Ut),f=vt(r.soft,-Ut),h=vt(n.mode,0),u=vt(r.mode,0),d=e-i,x=Yi(d),E=On(on(i),on(e)),m=Yi(E),p=on(m-x);(d<1e-24||p>10)&&(d=0,(i==0||e==0)&&(d=1e-24,h==2&&c!=Ut&&(s=0),u==2&&f!=-Ut&&(a=0)));let T=d||E||1e3,P=Yi(T),S=ws(10,jn(P)),C=T*(d==0?i==0?.1:1:s),y=Nt(Jh(i-C,S/10),24),I=i>=c&&(h==1||h==3&&y<=c||h==2&&y>=c)?c:Ut,v=On(o,y<I&&i>=I?I:_i(I,y)),A=T*(d==0?e==0?.1:1:a),F=Nt(Do(e+A,S/10),24),D=e<=f&&(u==1||u==3&&F>=f||u==2&&F<=f)?f:-Ut,z=_i(l,F>D&&e<=D?D:On(D,F));return v==z&&v==0&&(z=100),[v,z]}const Sm=new Intl.NumberFormat(Ns?pm.language:"en-US"),$c=i=>Sm.format(i),ei=Math,ao=ei.PI,on=ei.abs,jn=ei.floor,an=ei.round,si=ei.ceil,_i=ei.min,On=ei.max,ws=ei.pow,of=ei.sign,Yi=ei.log10,qh=ei.log2,Mm=(i,e=1)=>ei.sinh(i)*e,tl=(i,e=1)=>ei.asinh(i/e),Ut=1/0;function lf(i){return(Yi((i^i>>31)-(i>>31))|0)+1}function ql(i,e,t){return _i(On(i,e),t)}function Kh(i){return typeof i=="function"}function pt(i){return Kh(i)?i:()=>i}const ym=()=>{},$h=i=>i,Zh=(i,e)=>e,Em=i=>null,cf=i=>!0,uf=(i,e)=>i==e,bm=/\.\d*?(?=9{6,}|0{6,})/gm,zr=i=>{if(Qh(i)||_r.has(i))return i;const e=`${i}`,t=e.match(bm);if(t==null)return i;let n=t[0].length-1;if(e.indexOf("e-")!=-1){let[r,s]=e.split("e");return+`${zr(r)}e${s}`}return Nt(i,n)};function Pr(i,e){return zr(Nt(zr(i/e))*e)}function Do(i,e){return zr(si(zr(i/e))*e)}function Jh(i,e){return zr(jn(zr(i/e))*e)}function Nt(i,e=0){if(Qh(i))return i;let t=10**e,n=i*t*(1+Number.EPSILON);return an(n)/t}const _r=new Map;function jh(i){return((""+i).split(".")[1]||"").length}function ga(i,e,t,n){let r=[],s=n.map(jh);for(let a=e;a<t;a++){let o=on(a),l=Nt(ws(i,a),o);for(let c=0;c<n.length;c++){let f=i==10?+`${n[c]}e${a}`:n[c]*l,h=(a>=0?0:o)+(a>=s[c]?0:s[c]),u=i==10?f:Nt(f,h);r.push(u),_r.set(u,h)}}return r}const ma={},Zc=[],Rs=[null,null],fr=Array.isArray,Qh=Number.isInteger,Tm=i=>i===void 0;function ff(i){return typeof i=="string"}function Lo(i){let e=!1;if(i!=null){let t=i.constructor;e=t==null||t==Object}return e}function Am(i){return i!=null&&typeof i=="object"}const wm=Object.getPrototypeOf(Uint8Array),ed="__proto__";function Cs(i,e=Lo){let t;if(fr(i)){let n=i.find(r=>r!=null);if(fr(n)||e(n)){t=Array(i.length);for(let r=0;r<i.length;r++)t[r]=Cs(i[r],e)}else t=i.slice()}else if(i instanceof wm)t=i.slice();else if(e(i)){t={};for(let n in i)n!=ed&&(t[n]=Cs(i[n],e))}else t=i;return t}function nn(i){let e=arguments;for(let t=1;t<e.length;t++){let n=e[t];for(let r in n)r!=ed&&(Lo(i[r])?nn(i[r],Cs(n[r])):i[r]=Cs(n[r]))}return i}const Rm=0,Cm=1,Pm=2;function Dm(i,e,t){for(let n=0,r,s=-1;n<e.length;n++){let a=e[n];if(a>s){for(r=a-1;r>=0&&i[r]==null;)i[r--]=null;for(r=a+1;r<t&&i[r]==null;)i[s=r++]=null}}}function Lm(i,e){if(Nm(i)){let a=i[0].slice();for(let o=1;o<i.length;o++)a.push(...i[o].slice(1));return Fm(a[0])||(a=Um(a)),a}let t=new Set;for(let a=0;a<i.length;a++){let l=i[a][0],c=l.length;for(let f=0;f<c;f++)t.add(l[f])}let n=[Array.from(t).sort((a,o)=>a-o)],r=n[0].length,s=new Map;for(let a=0;a<r;a++)s.set(n[0][a],a);for(let a=0;a<i.length;a++){let o=i[a],l=o[0];for(let c=1;c<o.length;c++){let f=o[c],h=Array(r).fill(void 0),u=e?e[a][c]:Cm,d=[];for(let x=0;x<f.length;x++){let E=f[x],m=s.get(l[x]);E===null?u!=Rm&&(h[m]=E,u==Pm&&d.push(m)):h[m]=E}Dm(h,d,r),n.push(h)}}return n}const Im=typeof queueMicrotask>"u"?i=>Promise.resolve().then(i):queueMicrotask;function Um(i){let e=i[0],t=e.length,n=Array(t);for(let s=0;s<n.length;s++)n[s]=s;n.sort((s,a)=>e[s]-e[a]);let r=[];for(let s=0;s<i.length;s++){let a=i[s],o=Array(t);for(let l=0;l<t;l++)o[l]=a[n[l]];r.push(o)}return r}function Nm(i){let e=i[0][0],t=e.length;for(let n=1;n<i.length;n++){let r=i[n][0];if(r.length!=t)return!1;if(r!=e){for(let s=0;s<t;s++)if(r[s]!=e[s])return!1}}return!0}function Fm(i,e=100){const t=i.length;if(t<=1)return!0;let n=0,r=t-1;for(;n<=r&&i[n]==null;)n++;for(;r>=n&&i[r]==null;)r--;if(r<=n)return!0;const s=On(1,jn((r-n+1)/e));for(let a=i[n],o=n+s;o<=r;o+=s){const l=i[o];if(l!=null){if(l<=a)return!1;a=l}}return!0}const td=["January","February","March","April","May","June","July","August","September","October","November","December"],nd=["Sunday","Monday","Tuesday","Wednesday","Thursday","Friday","Saturday"];function id(i){return i.slice(0,3)}const Om=nd.map(id),Bm=td.map(id),zm={MMMM:td,MMM:Bm,WWWW:nd,WWW:Om};function Js(i){return(i<10?"0":"")+i}function km(i){return(i<10?"00":i<100?"0":"")+i}const Vm={YYYY:i=>i.getFullYear(),YY:i=>(i.getFullYear()+"").slice(2),MMMM:(i,e)=>e.MMMM[i.getMonth()],MMM:(i,e)=>e.MMM[i.getMonth()],MM:i=>Js(i.getMonth()+1),M:i=>i.getMonth()+1,DD:i=>Js(i.getDate()),D:i=>i.getDate(),WWWW:(i,e)=>e.WWWW[i.getDay()],WWW:(i,e)=>e.WWW[i.getDay()],HH:i=>Js(i.getHours()),H:i=>i.getHours(),h:i=>{let e=i.getHours();return e==0?12:e>12?e-12:e},AA:i=>i.getHours()>=12?"PM":"AM",aa:i=>i.getHours()>=12?"pm":"am",a:i=>i.getHours()>=12?"p":"a",mm:i=>Js(i.getMinutes()),m:i=>i.getMinutes(),ss:i=>Js(i.getSeconds()),s:i=>i.getSeconds(),fff:i=>km(i.getMilliseconds())};function Jc(i,e){e=e||zm;let t=[],n=/\{([a-z]+)\}|[^{]+/gi,r;for(;r=n.exec(i);)t.push(r[0][0]=="{"?Vm[r[1]]:r[0]);return s=>{let a="";for(let o=0;o<t.length;o++)a+=typeof t[o]=="string"?t[o]:t[o](s,e);return a}}const Gm=new Intl.DateTimeFormat().resolvedOptions().timeZone;function Hm(i,e){let t;return e=="UTC"||e=="Etc/UTC"?t=new Date(+i+i.getTimezoneOffset()*6e4):e==Gm?t=i:(t=new Date(i.toLocaleString("en-US",{timeZone:e})),t.setMilliseconds(i.getMilliseconds())),t}const rd=i=>i%1==0,xo=[1,2,2.5,5],Wm=ga(10,-32,0,xo),sd=ga(10,0,32,xo),Xm=sd.filter(rd),Dr=Wm.concat(sd),jc=`
`,ad="{YYYY}",hf=jc+ad,od="{M}/{D}",ua=jc+od,Ca=ua+"/{YY}",ld="{aa}",Ym="{h}:{mm}",Ss=Ym+ld,df=jc+Ss,pf=":{ss}",At=null;function cd(i){let e=i*1e3,t=e*60,n=t*60,r=n*24,s=r*30,a=r*365,l=(i==1?ga(10,0,3,xo).filter(rd):ga(10,-3,0,xo)).concat([e,e*5,e*10,e*15,e*30,t,t*5,t*10,t*15,t*30,n,n*2,n*3,n*4,n*6,n*8,n*12,r,r*2,r*3,r*4,r*5,r*6,r*7,r*8,r*9,r*10,r*15,s,s*2,s*3,s*4,s*6,a,a*2,a*5,a*10,a*25,a*50,a*100]);const c=[[a,ad,At,At,At,At,At,At,1],[r*28,"{MMM}",hf,At,At,At,At,At,1],[r,od,hf,At,At,At,At,At,1],[n,"{h}"+ld,Ca,At,ua,At,At,At,1],[t,Ss,Ca,At,ua,At,At,At,1],[e,pf,Ca+" "+Ss,At,ua+" "+Ss,At,df,At,1],[i,pf+".{fff}",Ca+" "+Ss,At,ua+" "+Ss,At,df,At,1]];function f(h){return(u,d,x,E,m,p)=>{let T=[],P=m>=a,S=m>=s&&m<a,C=h(x),y=Nt(C*i,3),I=nl(C.getFullYear(),P?0:C.getMonth(),S||P?1:C.getDate()),v=Nt(I*i,3);if(S||P){let A=S?m/s:0,F=P?m/a:0,D=y==v?y:Nt(nl(I.getFullYear()+F,I.getMonth()+A,1)*i,3),z=new Date(an(D/i)),O=z.getFullYear(),k=z.getMonth();for(let L=0;D<=E;L++){let N=nl(O+F*L,k+A*L,1),B=N-h(Nt(N*i,3));D=Nt((+N+B)*i,3),D<=E&&T.push(D)}}else{let A=m>=r?r:m,F=jn(x)-jn(y),D=v+F+Do(y-v,A);T.push(D);let z=h(D),O=z.getHours()+z.getMinutes()/t+z.getSeconds()/n,k=m/n,L=u.axes[d]._space,N=p/L;for(;D=Nt(D+m,i==1?0:3),!(D>E);)if(k>1){let B=jn(Nt(O+k,6))%24,ae=h(D).getHours()-B;ae>1&&(ae=-1),D-=ae*n,O=(O+k)%24;let fe=T[T.length-1];Nt((D-fe)/m,3)*N>=.7&&T.push(D)}else T.push(D)}return T}}return[l,c,f]}const[qm,Km,$m]=cd(1),[Zm,Jm,jm]=cd(.001);ga(2,-53,53,[1]);function mf(i,e){return i.map(t=>t.map((n,r)=>r==0||r==8||n==null?n:e(r==1||t[8]==0?n:t[1]+n)))}function gf(i,e){return(t,n,r,s,a)=>{let o=e.find(x=>a>=x[0])||e[e.length-1],l,c,f,h,u,d;return n.map(x=>{let E=i(x),m=E.getFullYear(),p=E.getMonth(),T=E.getDate(),P=E.getHours(),S=E.getMinutes(),C=E.getSeconds(),y=m!=l&&o[2]||p!=c&&o[3]||T!=f&&o[4]||P!=h&&o[5]||S!=u&&o[6]||C!=d&&o[7]||o[1];return l=m,c=p,f=T,h=P,u=S,d=C,y(E)})}}function Qm(i,e){let t=Jc(e);return(n,r,s,a,o)=>r.map(l=>t(i(l)))}function nl(i,e,t){return new Date(i,e,t)}function _f(i,e){return e(i)}const eg="{YYYY}-{MM}-{DD} {h}:{mm}{aa}";function xf(i,e){return(t,n,r,s)=>s==null?Xc:e(i(n))}function tg(i,e){let t=i.series[e];return t.width?t.stroke(i,e):t.points.width?t.points.stroke(i,e):null}function ng(i,e){return i.series[e].fill(i,e)}const ig={show:!0,live:!0,isolate:!1,mount:ym,markers:{show:!0,width:2,stroke:tg,fill:ng,dash:"solid"},idx:null,idxs:null,values:[]};function rg(i,e){let t=i.cursor.points,n=ri(),r=t.size(i,e);Vt(n,la,r),Vt(n,ca,r);let s=r/-2;Vt(n,"marginLeft",s),Vt(n,"marginTop",s);let a=t.width(i,e,r);return a&&Vt(n,"borderWidth",a),n}function sg(i,e){let t=i.series[e].points;return t._fill||t._stroke}function ag(i,e){let t=i.series[e].points;return t._stroke||t._fill}function og(i,e){return i.series[e].points.size}const il=[0,0];function lg(i,e,t){return il[0]=e,il[1]=t,il}function Pa(i,e,t,n=!0){return r=>{r.button==0&&(!n||r.target==e)&&t(r)}}function rl(i,e,t,n=!0){return r=>{(!n||r.target==e)&&t(r)}}const cg={show:!0,x:!0,y:!0,lock:!1,move:lg,points:{one:!1,show:rg,size:og,width:0,stroke:ag,fill:sg},bind:{mousedown:Pa,mouseup:Pa,click:Pa,dblclick:Pa,mousemove:rl,mouseleave:rl,mouseenter:rl},drag:{setScale:!0,x:!0,y:!1,dist:0,uni:null,click:(i,e)=>{e.stopPropagation(),e.stopImmediatePropagation()},_x:!1,_y:!1},focus:{dist:(i,e,t,n,r)=>n-r,prox:-1,bias:0},hover:{skip:[void 0],prox:null,bias:0},left:-10,top:-10,idx:null,dataIdx:null,idxs:null,event:null},ud={show:!0,stroke:"rgba(0,0,0,0.07)",width:2},Qc=nn({},ud,{filter:Zh}),fd=nn({},Qc,{size:10}),hd=nn({},ud,{show:!1}),eu='12px system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"',dd="bold "+eu,pd=1.5,vf={show:!0,scale:"x",stroke:Wc,space:50,gap:5,alignTo:1,size:50,labelGap:0,labelSize:30,labelFont:dd,side:2,grid:Qc,ticks:fd,border:hd,font:eu,lineGap:pd,rotate:0},ug="Value",fg="Time",Sf={show:!0,scale:"x",auto:!1,sorted:1,min:Ut,max:-Ut,idxs:[]};function hg(i,e,t,n,r){return e.map(s=>s==null?"":$c(s))}function dg(i,e,t,n,r,s,a){let o=[],l=_r.get(r)||0;t=a?t:Nt(Do(t,r),l);for(let c=t;c<=n;c=Nt(c+r,l))o.push(Object.is(c,-0)?0:c);return o}function Kl(i,e,t,n,r,s,a){const o=[],l=i.scales[i.axes[e].scale].log,c=l==10?Yi:qh,f=jn(c(t));r=ws(l,f),l==10&&(r=Dr[gi(r,Dr)]);let h=t,u=r*l;l==10&&(u=Dr[gi(u,Dr)]);do o.push(h),h=h+r,l==10&&!_r.has(h)&&(h=Nt(h,_r.get(r))),h>=u&&(r=h,u=r*l,l==10&&(u=Dr[gi(u,Dr)]));while(h<=n);return o}function pg(i,e,t,n,r,s,a){let l=i.scales[i.axes[e].scale].asinh,c=n>l?Kl(i,e,On(l,t),n,r):[l],f=n>=0&&t<=0?[0]:[];return(t<-l?Kl(i,e,On(l,-n),-t,r):[l]).reverse().map(u=>-u).concat(f,c)}const md=/./,mg=/[12357]/,gg=/[125]/,Mf=/1/,$l=(i,e,t,n)=>i.map((r,s)=>e==4&&r==0||s%n==0&&t.test(r.toExponential()[r<0?1:0])?r:null);function _g(i,e,t,n,r){let s=i.axes[t],a=s.scale,o=i.scales[a],l=i.valToPos,c=s._space,f=l(10,a),h=l(9,a)-f>=c?md:l(7,a)-f>=c?mg:l(5,a)-f>=c?gg:Mf;if(h==Mf){let u=on(l(1,a)-f);if(u<c)return $l(e.slice().reverse(),o.distr,h,si(c/u)).reverse()}return $l(e,o.distr,h,1)}function xg(i,e,t,n,r){let s=i.axes[t],a=s.scale,o=s._space,l=i.valToPos,c=on(l(1,a)-l(2,a));return c<o?$l(e.slice().reverse(),3,md,si(o/c)).reverse():e}function vg(i,e,t,n){return n==null?Xc:e==null?"":$c(e)}const yf={show:!0,scale:"y",stroke:Wc,space:30,gap:5,alignTo:1,size:50,labelGap:0,labelSize:30,labelFont:dd,side:3,grid:Qc,ticks:fd,border:hd,font:eu,lineGap:pd,rotate:0};function Sg(i,e){let t=3+(i||1)*2;return Nt(t*e,3)}function Mg(i,e){let{scale:t,idxs:n}=i.series[0],r=i._data[0],s=i.valToPos(r[n[0]],t,!0),a=i.valToPos(r[n[1]],t,!0),o=on(a-s),l=i.series[e],c=o/(l.points.space*bt);return n[1]-n[0]<=c}const Ef={scale:null,auto:!0,sorted:0,min:Ut,max:-Ut},gd=(i,e,t,n,r)=>r,bf={show:!0,auto:!0,sorted:0,gaps:gd,alpha:1,facets:[nn({},Ef,{scale:"x"}),nn({},Ef,{scale:"y"})]},Tf={scale:"y",auto:!0,sorted:0,show:!0,spanGaps:!1,gaps:gd,alpha:1,points:{show:Mg,filter:null},values:null,min:Ut,max:-Ut,idxs:[],path:null,clip:null};function yg(i,e,t,n,r){return t/10}const _d={time:qp,auto:!0,distr:1,log:10,asinh:1,min:null,max:null,dir:1,ori:0},Eg=nn({},_d,{time:!1,ori:1}),Af={};function xd(i,e){let t=Af[i];return t||(t={key:i,plots:[],sub(n){t.plots.push(n)},unsub(n){t.plots=t.plots.filter(r=>r!=n)},pub(n,r,s,a,o,l,c){for(let f=0;f<t.plots.length;f++)t.plots[f]!=r&&t.plots[f].pub(n,r,s,a,o,l,c)}},i!=null&&(Af[i]=t)),t}const Ps=1,Zl=2;function Hr(i,e,t){const n=i.mode,r=i.series[e],s=n==2?i._data[e]:i._data,a=i.scales,o=i.bbox;let l=s[0],c=n==2?s[1]:s[e],f=n==2?a[r.facets[0].scale]:a[i.series[0].scale],h=n==2?a[r.facets[1].scale]:a[r.scale],u=o.left,d=o.top,x=o.width,E=o.height,m=i.valToPosH,p=i.valToPosV;return f.ori==0?t(r,l,c,f,h,m,p,u,d,x,E,Uo,Fs,Fo,Sd,yd):t(r,l,c,f,h,p,m,d,u,E,x,No,Os,iu,Md,Ed)}function tu(i,e){let t=0,n=0,r=vt(i.bands,Zc);for(let s=0;s<r.length;s++){let a=r[s];a.series[0]==e?t=a.dir:a.series[1]==e&&(a.dir==1?n|=1:n|=2)}return[t,n==1?-1:n==2?1:n==3?2:0]}function bg(i,e,t,n,r){let s=i.mode,a=i.series[e],o=s==2?a.facets[1].scale:a.scale,l=i.scales[o];return r==-1?l.min:r==1?l.max:l.distr==3?l.dir==1?l.min:l.max:0}function qi(i,e,t,n,r,s){return Hr(i,e,(a,o,l,c,f,h,u,d,x,E,m)=>{let p=a.pxRound;const T=c.dir*(c.ori==0?1:-1),P=c.ori==0?Fs:Os;let S,C;T==1?(S=t,C=n):(S=n,C=t);let y=p(h(o[S],c,E,d)),I=p(u(l[S],f,m,x)),v=p(h(o[C],c,E,d)),A=p(u(s==1?f.max:f.min,f,m,x)),F=new Path2D(r);return P(F,v,A),P(F,y,A),P(F,y,I),F})}function Io(i,e,t,n,r,s){let a=null;if(i.length>0){a=new Path2D;const o=e==0?Fo:iu;let l=t;for(let h=0;h<i.length;h++){let u=i[h];if(u[1]>u[0]){let d=u[0]-l;d>0&&o(a,l,n,d,n+s),l=u[1]}}let c=t+r-l,f=10;c>0&&o(a,l,n-f/2,c,n+s+f)}return a}function Tg(i,e,t){let n=i[i.length-1];n&&n[0]==e?n[1]=t:i.push([e,t])}function nu(i,e,t,n,r,s,a){let o=[],l=i.length;for(let c=r==1?t:n;c>=t&&c<=n;c+=r)if(e[c]===null){let h=c,u=c;if(r==1)for(;++c<=n&&e[c]===null;)u=c;else for(;--c>=t&&e[c]===null;)u=c;let d=s(i[h]),x=u==h?d:s(i[u]),E=h-r;d=a<=0&&E>=0&&E<l?s(i[E]):d;let p=u+r;x=a>=0&&p>=0&&p<l?s(i[p]):x,x>=d&&o.push([d,x])}return o}function wf(i){return i==0?$h:i==1?an:e=>Pr(e,i)}function vd(i){let e=i==0?Uo:No,t=i==0?(r,s,a,o,l,c)=>{r.arcTo(s,a,o,l,c)}:(r,s,a,o,l,c)=>{r.arcTo(a,s,l,o,c)},n=i==0?(r,s,a,o,l)=>{r.rect(s,a,o,l)}:(r,s,a,o,l)=>{r.rect(a,s,l,o)};return(r,s,a,o,l,c=0,f=0)=>{c==0&&f==0?n(r,s,a,o,l):(c=_i(c,o/2,l/2),f=_i(f,o/2,l/2),e(r,s+c,a),t(r,s+o,a,s+o,a+l,c),t(r,s+o,a+l,s,a+l,f),t(r,s,a+l,s,a,f),t(r,s,a,s+o,a,c),r.closePath())}}const Uo=(i,e,t)=>{i.moveTo(e,t)},No=(i,e,t)=>{i.moveTo(t,e)},Fs=(i,e,t)=>{i.lineTo(e,t)},Os=(i,e,t)=>{i.lineTo(t,e)},Fo=vd(0),iu=vd(1),Sd=(i,e,t,n,r,s)=>{i.arc(e,t,n,r,s)},Md=(i,e,t,n,r,s)=>{i.arc(t,e,n,r,s)},yd=(i,e,t,n,r,s,a)=>{i.bezierCurveTo(e,t,n,r,s,a)},Ed=(i,e,t,n,r,s,a)=>{i.bezierCurveTo(t,e,r,n,a,s)};function bd(i){return(e,t,n,r,s)=>Hr(e,t,(a,o,l,c,f,h,u,d,x,E,m)=>{let{pxRound:p,points:T}=a,P,S;c.ori==0?(P=Uo,S=Sd):(P=No,S=Md);const C=Nt(T.width*bt,3);let y=(T.size-T.width)/2*bt,I=Nt(y*2,3),v=new Path2D,A=new Path2D,{left:F,top:D,width:z,height:O}=e.bbox;Fo(A,F-I,D-I,z+I*2,O+I*2);const k=L=>{if(l[L]!=null){let N=p(h(o[L],c,E,d)),B=p(u(l[L],f,m,x));P(v,N+y,B),S(v,N,B,y,0,ao*2)}};if(s)s.forEach(k);else for(let L=n;L<=r;L++)k(L);return{stroke:C>0?v:null,fill:v,clip:A,flags:Ps|Zl}})}function Td(i){return(e,t,n,r,s,a)=>{n!=r&&(s!=n&&a!=n&&i(e,t,n),s!=r&&a!=r&&i(e,t,r),i(e,t,a))}}const Ag=Td(Fs),wg=Td(Os);function Ad(i){const e=vt(i==null?void 0:i.alignGaps,0);return(t,n,r,s)=>Hr(t,n,(a,o,l,c,f,h,u,d,x,E,m)=>{[r,s]=Co(l,r,s);let p=a.pxRound,T=O=>p(h(O,c,E,d)),P=O=>p(u(O,f,m,x)),S,C;c.ori==0?(S=Fs,C=Ag):(S=Os,C=wg);const y=c.dir*(c.ori==0?1:-1),I={stroke:new Path2D,fill:null,clip:null,band:null,gaps:null,flags:Ps},v=I.stroke;let A=!1;if(s-r>=E*4){let O=le=>t.posToVal(le,c.key,!0),k=null,L=null,N,B,j,ne=T(o[y==1?r:s]),ae=T(o[r]),fe=T(o[s]),re=O(y==1?ae+1:fe-1);for(let le=y==1?r:s;le>=r&&le<=s;le+=y){let Ze=o[le],Q=(y==1?Ze<re:Ze>re)?ne:T(Ze),ue=l[le];Q==ne?ue!=null?(B=ue,k==null?(S(v,Q,P(B)),N=k=L=B):B<k?k=B:B>L&&(L=B)):ue===null&&(A=!0):(k!=null&&C(v,ne,P(k),P(L),P(N),P(B)),ue!=null?(B=ue,S(v,Q,P(B)),k=L=N=B):(k=L=null,ue===null&&(A=!0)),ne=Q,re=O(ne+y))}k!=null&&k!=L&&j!=ne&&C(v,ne,P(k),P(L),P(N),P(B))}else for(let O=y==1?r:s;O>=r&&O<=s;O+=y){let k=l[O];k===null?A=!0:k!=null&&S(v,T(o[O]),P(k))}let[D,z]=tu(t,n);if(a.fill!=null||D!=0){let O=I.fill=new Path2D(v),k=a.fillTo(t,n,a.min,a.max,D),L=P(k),N=T(o[r]),B=T(o[s]);y==-1&&([B,N]=[N,B]),S(O,B,L),S(O,N,L)}if(!a.spanGaps){let O=[];A&&O.push(...nu(o,l,r,s,y,T,e)),I.gaps=O=a.gaps(t,n,r,s,O),I.clip=Io(O,c.ori,d,x,E,m)}return z!=0&&(I.band=z==2?[qi(t,n,r,s,v,-1),qi(t,n,r,s,v,1)]:qi(t,n,r,s,v,z)),I})}function Rg(i){const e=vt(i.align,1),t=vt(i.ascDesc,!1),n=vt(i.alignGaps,0),r=vt(i.extend,!1);return(s,a,o,l)=>Hr(s,a,(c,f,h,u,d,x,E,m,p,T,P)=>{[o,l]=Co(h,o,l);let S=c.pxRound,{left:C,width:y}=s.bbox,I=ae=>S(x(ae,u,T,m)),v=ae=>S(E(ae,d,P,p)),A=u.ori==0?Fs:Os;const F={stroke:new Path2D,fill:null,clip:null,band:null,gaps:null,flags:Ps},D=F.stroke,z=u.dir*(u.ori==0?1:-1);let O=v(h[z==1?o:l]),k=I(f[z==1?o:l]),L=k,N=k;r&&e==-1&&(N=C,A(D,N,O)),A(D,k,O);for(let ae=z==1?o:l;ae>=o&&ae<=l;ae+=z){let fe=h[ae];if(fe==null)continue;let re=I(f[ae]),le=v(fe);e==1?A(D,re,O):A(D,L,le),A(D,re,le),O=le,L=re}let B=L;r&&e==1&&(B=C+y,A(D,B,O));let[j,ne]=tu(s,a);if(c.fill!=null||j!=0){let ae=F.fill=new Path2D(D),fe=c.fillTo(s,a,c.min,c.max,j),re=v(fe);A(ae,B,re),A(ae,N,re)}if(!c.spanGaps){let ae=[];ae.push(...nu(f,h,o,l,z,I,n));let fe=c.width*bt/2,re=t||e==1?fe:-fe,le=t||e==-1?-fe:fe;ae.forEach(Ze=>{Ze[0]+=re,Ze[1]+=le}),F.gaps=ae=c.gaps(s,a,o,l,ae),F.clip=Io(ae,u.ori,m,p,T,P)}return ne!=0&&(F.band=ne==2?[qi(s,a,o,l,D,-1),qi(s,a,o,l,D,1)]:qi(s,a,o,l,D,ne)),F})}function Rf(i,e,t,n,r,s,a=Ut){if(i.length>1){let o=null;for(let l=0,c=1/0;l<i.length;l++)if(e[l]!==void 0){if(o!=null){let f=on(i[l]-i[o]);f<c&&(c=f,a=on(t(i[l],n,r,s)-t(i[o],n,r,s)))}o=l}}return a}function Cg(i){i=i||ma;const e=vt(i.size,[.6,Ut,1]),t=i.align||0,n=i.gap||0;let r=i.radius;r=r==null?[0,0]:typeof r=="number"?[r,0]:r;const s=pt(r),a=1-e[0],o=vt(e[1],Ut),l=vt(e[2],1),c=vt(i.disp,ma),f=vt(i.each,d=>{}),{fill:h,stroke:u}=c;return(d,x,E,m)=>Hr(d,x,(p,T,P,S,C,y,I,v,A,F,D)=>{let z=p.pxRound,O=t,k=n*bt,L=o*bt,N=l*bt,B,j;S.ori==0?[B,j]=s(d,x):[j,B]=s(d,x);const ne=S.dir*(S.ori==0?1:-1);let ae=S.ori==0?Fo:iu,fe=S.ori==0?f:(pe,w,_,Y,$,ee,ge)=>{f(pe,w,_,$,Y,ge,ee)},re=vt(d.bands,Zc).find(pe=>pe.series[0]==x),le=re!=null?re.dir:0,Ze=p.fillTo(d,x,p.min,p.max,le),Ke=z(I(Ze,C,D,A)),Q,ue,he,Be=F,Ue=z(p.width*bt),ze=!1,St=null,je=null,ht=null,ot=null;h!=null&&(Ue==0||u!=null)&&(ze=!0,St=h.values(d,x,E,m),je=new Map,new Set(St).forEach(pe=>{pe!=null&&je.set(pe,new Path2D)}),Ue>0&&(ht=u.values(d,x,E,m),ot=new Map,new Set(ht).forEach(pe=>{pe!=null&&ot.set(pe,new Path2D)})));let{x0:rt,size:Ft}=c;if(rt!=null&&Ft!=null){O=1,T=rt.values(d,x,E,m),rt.unit==2&&(T=T.map(_=>d.posToVal(v+_*F,S.key,!0)));let pe=Ft.values(d,x,E,m);Ft.unit==2?ue=pe[0]*F:ue=y(pe[0],S,F,v)-y(0,S,F,v),Be=Rf(T,P,y,S,F,v,Be),he=Be-ue+k}else Be=Rf(T,P,y,S,F,v,Be),he=Be*a+k,ue=Be-he;he<1&&(he=0),Ue>=ue/2&&(Ue=0),he<5&&(z=$h);let Gt=he>0,Ot=Be-he-(Gt?Ue:0);ue=z(ql(Ot,N,L)),Q=(O==0?ue/2:O==ne?0:ue)-O*ne*((O==0?k/2:0)+(Gt?Ue/2:0));const _t={stroke:null,fill:null,clip:null,band:null,gaps:null,flags:0},wt=ze?null:new Path2D;let Tt=null;if(re!=null)Tt=d.data[re.series[1]];else{let{y0:pe,y1:w}=c;pe!=null&&w!=null&&(P=w.values(d,x,E,m),Tt=pe.values(d,x,E,m))}let H=B*ue,Xe=j*ue;for(let pe=ne==1?E:m;pe>=E&&pe<=m;pe+=ne){let w=P[pe];if(w==null)continue;if(Tt!=null){let ie=Tt[pe]??0;if(w-ie==0)continue;Ke=I(ie,C,D,A)}let _=S.distr!=2||c!=null?T[pe]:pe,Y=y(_,S,F,v),$=I(vt(w,Ze),C,D,A),ee=z(Y-Q),ge=z(On($,Ke)),_e=z(_i($,Ke)),te=ge-_e;if(w!=null){let ie=w<0?Xe:H,xe=w<0?H:Xe;ze?(Ue>0&&ht[pe]!=null&&ae(ot.get(ht[pe]),ee,_e+jn(Ue/2),ue,On(0,te-Ue),ie,xe),St[pe]!=null&&ae(je.get(St[pe]),ee,_e+jn(Ue/2),ue,On(0,te-Ue),ie,xe)):ae(wt,ee,_e+jn(Ue/2),ue,On(0,te-Ue),ie,xe),fe(d,x,pe,ee-Ue/2,_e,ue+Ue,te)}}return Ue>0?_t.stroke=ze?ot:wt:ze||(_t._fill=p.width==0?p._fill:p._stroke??p._fill,_t.width=0),_t.fill=ze?je:wt,_t})}function Pg(i,e){const t=vt(e==null?void 0:e.alignGaps,0);return(n,r,s,a)=>Hr(n,r,(o,l,c,f,h,u,d,x,E,m,p)=>{[s,a]=Co(c,s,a);let T=o.pxRound,P=B=>T(u(B,f,m,x)),S=B=>T(d(B,h,p,E)),C,y,I;f.ori==0?(C=Uo,I=Fs,y=yd):(C=No,I=Os,y=Ed);const v=f.dir*(f.ori==0?1:-1);let A=P(l[v==1?s:a]),F=A,D=[],z=[];for(let B=v==1?s:a;B>=s&&B<=a;B+=v)if(c[B]!=null){let ne=l[B],ae=P(ne);D.push(F=ae),z.push(S(c[B]))}const O={stroke:i(D,z,C,I,y,T),fill:null,clip:null,band:null,gaps:null,flags:Ps},k=O.stroke;let[L,N]=tu(n,r);if(o.fill!=null||L!=0){let B=O.fill=new Path2D(k),j=o.fillTo(n,r,o.min,o.max,L),ne=S(j);I(B,F,ne),I(B,A,ne)}if(!o.spanGaps){let B=[];B.push(...nu(l,c,s,a,v,P,t)),O.gaps=B=o.gaps(n,r,s,a,B),O.clip=Io(B,f.ori,x,E,m,p)}return N!=0&&(O.band=N==2?[qi(n,r,s,a,k,-1),qi(n,r,s,a,k,1)]:qi(n,r,s,a,k,N)),O})}function Dg(i){return Pg(Lg,i)}function Lg(i,e,t,n,r,s){const a=i.length;if(a<2)return null;const o=new Path2D;if(t(o,i[0],e[0]),a==2)n(o,i[1],e[1]);else{let l=Array(a),c=Array(a-1),f=Array(a-1),h=Array(a-1);for(let u=0;u<a-1;u++)f[u]=e[u+1]-e[u],h[u]=i[u+1]-i[u],c[u]=f[u]/h[u];l[0]=c[0];for(let u=1;u<a-1;u++)c[u]===0||c[u-1]===0||c[u-1]>0!=c[u]>0?l[u]=0:(l[u]=3*(h[u-1]+h[u])/((2*h[u]+h[u-1])/c[u-1]+(h[u]+2*h[u-1])/c[u]),isFinite(l[u])||(l[u]=0));l[a-1]=c[a-2];for(let u=0;u<a-1;u++)r(o,i[u]+h[u]/3,e[u]+l[u]*h[u]/3,i[u+1]-h[u]/3,e[u+1]-l[u+1]*h[u]/3,i[u+1],e[u+1])}return o}const Jl=new Set;function Cf(){for(let i of Jl)i.syncRect(!0)}Ns&&(Br(hm,ys,Cf),Br(dm,ys,Cf,!0),Br(go,ys,()=>{An.pxRatio=bt}));const Ig=Ad(),Ug=bd();function Pf(i,e,t,n){return(n?[i[0],i[1]].concat(i.slice(2)):[i[0]].concat(i.slice(1))).map((s,a)=>jl(s,a,e,t))}function Ng(i,e){return i.map((t,n)=>n==0?{}:nn({},e,t))}function jl(i,e,t,n){return nn({},e==0?t:n,i)}function wd(i,e,t){return e==null?Rs:[e,t]}const Fg=wd;function Og(i,e,t){return e==null?Rs:_o(e,t,Kc,!0)}function Rd(i,e,t,n){return e==null?Rs:Po(e,t,i.scales[n].log,!1)}const Bg=Rd;function Cd(i,e,t,n){return e==null?Rs:qc(e,t,i.scales[n].log,!1)}const zg=Cd;function kg(i,e,t,n,r){let s=On(lf(i),lf(e)),a=e-i,o=gi(r/n*a,t);do{let l=t[o],c=n*l/a;if(c>=r&&s+(l<5?_r.get(l):0)<=17)return[l,c]}while(++o<t.length);return[0,0]}function Df(i){let e,t;return i=i.replace(/(\d+)px/,(n,r)=>(e=an((t=+r)*bt))+"px"),[i,e,t]}function Vg(i){i.show&&[i.font,i.labelFont].forEach(e=>{let t=Nt(e[2]*bt,1);e[0]=e[0].replace(/[0-9.]+px/,t+"px"),e[1]=t})}function An(i,e,t){const n={mode:vt(i.mode,1)},r=n.mode;function s(g,M,R,U){let V=M.valToPct(g);return U+R*(M.dir==-1?1-V:V)}function a(g,M,R,U){let V=M.valToPct(g);return U+R*(M.dir==-1?V:1-V)}function o(g,M,R,U){return M.ori==0?s(g,M,R,U):a(g,M,R,U)}n.valToPosH=s,n.valToPosV=a;let l=!1;n.status=0;const c=n.root=ri(Kp);if(i.id!=null&&(c.id=i.id),Zn(c,i.class),i.title){let g=ri(Jp,c);g.textContent=i.title}const f=mi("canvas"),h=n.ctx=f.getContext("2d"),u=ri(jp,c);Br("click",u,g=>{g.target===x&&(Bt!=$r||Wt!=Zr)&&Sn.click(n,g)},!0);const d=n.under=ri(Qp,u);u.appendChild(f);const x=n.over=ri(em,u);i=Cs(i);const E=+vt(i.pxAlign,1),m=wf(E);(i.plugins||[]).forEach(g=>{g.opts&&(i=g.opts(n,i)||i)});const p=i.ms||.001,T=n.series=r==1?Pf(i.series||[],Sf,Tf,!1):Ng(i.series||[null],bf),P=n.axes=Pf(i.axes||[],vf,yf,!0),S=n.scales={},C=n.bands=i.bands||[];C.forEach(g=>{g.fill=pt(g.fill||null),g.dir=vt(g.dir,-1)});const y=r==2?T[1].facets[0].scale:T[0].scale,I={axes:Pt,series:Rt},v=(i.drawOrder||["axes","series"]).map(g=>I[g]);function A(g){const M=g.distr==3?R=>Yi(R>0?R:g.clamp(n,R,g.min,g.max,g.key)):g.distr==4?R=>tl(R,g.asinh):g.distr==100?R=>g.fwd(R):R=>R;return R=>{let U=M(R),{_min:V,_max:K}=g,ce=K-V;return(U-V)/ce}}function F(g){let M=S[g];if(M==null){let R=(i.scales||ma)[g]||ma;if(R.from!=null){F(R.from);let U=nn({},S[R.from],R,{key:g});U.valToPct=A(U),S[g]=U}else{M=S[g]=nn({},g==y?_d:Eg,R),M.key=g;let U=M.time,V=M.range,K=fr(V);if((g!=y||r==2&&!U)&&(K&&(V[0]==null||V[1]==null)&&(V={min:V[0]==null?sf:{mode:1,hard:V[0],soft:V[0]},max:V[1]==null?sf:{mode:1,hard:V[1],soft:V[1]}},K=!1),!K&&Lo(V))){let ce=V;V=(me,ve,Pe)=>ve==null?Rs:_o(ve,Pe,ce)}M.range=pt(V||(U?Fg:g==y?M.distr==3?Bg:M.distr==4?zg:wd:M.distr==3?Rd:M.distr==4?Cd:Og)),M.auto=pt(K?!1:M.auto),M.clamp=pt(M.clamp||yg),M._min=M._max=null,M.valToPct=A(M)}}}F("x"),F("y"),r==1&&T.forEach(g=>{F(g.scale)}),P.forEach(g=>{F(g.scale)});for(let g in i.scales)F(g);const D=S[y],z=D.distr;let O,k;D.ori==0?(Zn(c,$p),O=s,k=a):(Zn(c,Zp),O=a,k=s);const L={};for(let g in S){let M=S[g];(M.min!=null||M.max!=null)&&(L[g]={min:M.min,max:M.max},M.min=M.max=null)}const N=i.tzDate||(g=>new Date(an(g/p))),B=i.fmtDate||Jc,j=p==1?$m(N):jm(N),ne=gf(N,mf(p==1?Km:Jm,B)),ae=xf(N,_f(eg,B)),fe=[],re=n.legend=nn({},ig,i.legend),le=n.cursor=nn({},cg,{drag:{y:r==2}},i.cursor),Ze=re.show,Ke=le.show,Q=re.markers;re.idxs=fe,Q.width=pt(Q.width),Q.dash=pt(Q.dash),Q.stroke=pt(Q.stroke),Q.fill=pt(Q.fill);let ue,he,Be,Ue=[],ze=[],St,je=!1,ht={};if(re.live){const g=T[1]?T[1].values:null;je=g!=null,St=je?g(n,1,0):{_:0};for(let M in St)ht[M]=Xc}if(Ze)if(ue=mi("table",am,c),Be=mi("tbody",null,ue),re.mount(n,ue),je){he=mi("thead",null,ue,Be);let g=mi("tr",null,he);mi("th",null,g);for(var ot in St)mi("th",Xu,g).textContent=ot}else Zn(ue,lm),re.live&&Zn(ue,om);const rt={show:!0},Ft={show:!1};function Gt(g,M){if(M==0&&(je||!re.live||r==2))return Rs;let R=[],U=mi("tr",cm,Be,Be.childNodes[M]);Zn(U,g.class),g.show||Zn(U,Lr);let V=mi("th",null,U);if(Q.show){let me=ri(um,V);if(M>0){let ve=Q.width(n,M);ve&&(me.style.border=ve+"px "+Q.dash(n,M)+" "+Q.stroke(n,M)),me.style.background=Q.fill(n,M)}}let K=ri(Xu,V);g.label instanceof HTMLElement?K.appendChild(g.label):K.textContent=g.label,M>0&&(Q.show||(K.style.color=g.width>0?Q.stroke(n,M):Q.fill(n,M)),_t("click",V,me=>{if(le._lock)return;Ne(me);let ve=T.indexOf(g);if((me.ctrlKey||me.metaKey)!=re.isolate){let Pe=T.some((Ie,Fe)=>Fe>0&&Fe!=ve&&Ie.show);T.forEach((Ie,Fe)=>{Fe>0&&Mi(Fe,Pe?Fe==ve?rt:Ft:rt,!0,en.setSeries)})}else Mi(ve,{show:!g.show},!0,en.setSeries)},!1),xn&&_t($u,V,me=>{le._lock||(Ne(me),Mi(T.indexOf(g),jr,!0,en.setSeries))},!1));for(var ce in St){let me=mi("td",fm,U);me.textContent="--",R.push(me)}return[U,R]}const Ot=new Map;function _t(g,M,R,U=!0){const V=Ot.get(M)||{},K=le.bind[g](n,M,R,U);K&&(Br(g,M,V[g]=K),Ot.set(M,V))}function wt(g,M,R){const U=Ot.get(M)||{};for(let V in U)(g==null||V==g)&&(Yl(V,M,U[V]),delete U[V]);g==null&&Ot.delete(M)}let Tt=0,H=0,Xe=0,pe=0,w=0,_=0,Y=w,$=_,ee=Xe,ge=pe,_e=0,te=0,ie=0,xe=0;n.bbox={};let Ve=!1,ye=!1,Se=!1,ke=!1,Ye=!1,qe=!1;function G(g,M,R){(R||g!=n.width||M!=n.height)&&Me(g,M),Dn(!1),Se=!0,ye=!0,Yr()}function Me(g,M){n.width=Tt=Xe=g,n.height=H=pe=M,w=_=0,de(),Ge();let R=n.bbox;_e=R.left=Pr(w*bt,.5),te=R.top=Pr(_*bt,.5),ie=R.width=Pr(Xe*bt,.5),xe=R.height=Pr(pe*bt,.5)}const oe=3;function Ee(){let g=!1,M=0;for(;!g;){M++;let R=Kt(M),U=Si(M);g=M==oe||R&&U,g||(Me(n.width,n.height),ye=!0)}}function Re({width:g,height:M}){G(g,M)}n.setSize=Re;function de(){let g=!1,M=!1,R=!1,U=!1;P.forEach((V,K)=>{if(V.show&&V._show){let{side:ce,_size:me}=V,ve=ce%2,Pe=V.label!=null?V.labelSize:0,Ie=me+Pe;Ie>0&&(ve?(Xe-=Ie,ce==3?(w+=Ie,U=!0):R=!0):(pe-=Ie,ce==0?(_+=Ie,g=!0):M=!0))}}),Rn[0]=g,Rn[1]=R,Rn[2]=M,Rn[3]=U,Xe-=Gn[1]+Gn[3],w+=Gn[3],pe-=Gn[2]+Gn[0],_+=Gn[0]}function Ge(){let g=w+Xe,M=_+pe,R=w,U=_;function V(K,ce){switch(K){case 1:return g+=ce,g-ce;case 2:return M+=ce,M-ce;case 3:return R-=ce,R+ce;case 0:return U-=ce,U+ce}}P.forEach((K,ce)=>{if(K.show&&K._show){let me=K.side;K._pos=V(me,K._size),K.label!=null&&(K._lpos=V(me,K.labelSize))}})}if(le.dataIdx==null){let g=le.hover,M=g.skip=new Set(g.skip??[]);M.add(void 0);let R=g.prox=pt(g.prox),U=g.bias??(g.bias=0);le.dataIdx=(V,K,ce,me)=>{if(K==0)return ce;let ve=ce,Pe=R(V,K,ce,me)??Ut,Ie=Pe>=0&&Pe<Ut,Fe=D.ori==0?Xe:pe,tt=le.left,Et=e[0],xt=e[K];if(M.has(xt[ce])){ve=null;let ct=null,Qe=null,We;if(U==0||U==-1)for(We=ce;ct==null&&We-- >0;)M.has(xt[We])||(ct=We);if(U==0||U==1)for(We=ce;Qe==null&&We++<xt.length;)M.has(xt[We])||(Qe=We);if(ct!=null||Qe!=null)if(Ie){let kt=ct==null?-1/0:O(Et[ct],D,Fe,0),Zt=Qe==null?1/0:O(Et[Qe],D,Fe,0),pn=tt-kt,Dt=Zt-tt;pn<=Dt?pn<=Pe&&(ve=ct):Dt<=Pe&&(ve=Qe)}else ve=Qe==null?ct:ct==null?Qe:ce-ct<=Qe-ce?ct:Qe}else Ie&&on(tt-O(Et[ce],D,Fe,0))>Pe&&(ve=null);return ve}}const Ne=g=>{le.event=g};le.idxs=fe,le._lock=!1;let at=le.points;at.show=pt(at.show),at.size=pt(at.size),at.stroke=pt(at.stroke),at.width=pt(at.width),at.fill=pt(at.fill);const lt=n.focus=nn({},i.focus||{alpha:.3},le.focus),xn=lt.prox>=0,vn=xn&&at.one;let zn=[],ji=[],Bi=[];function Wr(g,M){let R=at.show(n,M);if(R instanceof HTMLElement)return Zn(R,sm),Zn(R,g.class),Ti(R,-10,-10,Xe,pe),x.insertBefore(R,zn[M]),R}function ya(g,M){if(r==1||M>0){let R=r==1&&S[g.scale].time,U=g.value;g.value=R?ff(U)?xf(N,_f(U,B)):U||ae:U||vg,g.label=g.label||(R?fg:ug)}if(vn||M>0){g.width=g.width==null?1:g.width,g.paths=g.paths||Ig||Em,g.fillTo=pt(g.fillTo||bg),g.pxAlign=+vt(g.pxAlign,E),g.pxRound=wf(g.pxAlign),g.stroke=pt(g.stroke||null),g.fill=pt(g.fill||null),g._stroke=g._fill=g._paths=g._focus=null;let R=Sg(On(1,g.width),1),U=g.points=nn({},{size:R,width:On(1,R*.2),stroke:g.stroke,space:R*2,paths:Ug,_stroke:null,_fill:null},g.points);U.show=pt(U.show),U.filter=pt(U.filter),U.fill=pt(U.fill),U.stroke=pt(U.stroke),U.paths=pt(U.paths),U.pxAlign=g.pxAlign}if(Ze){let R=Gt(g,M);Ue.splice(M,0,R[0]),ze.splice(M,0,R[1]),re.values.push(null)}if(Ke){fe.splice(M,0,null);let R=null;vn?M==0&&(R=Wr(g,M)):M>0&&(R=Wr(g,M)),zn.splice(M,0,R),ji.splice(M,0,0),Bi.splice(M,0,0)}dn("addSeries",M)}function Ea(g,M){M=M??T.length,g=r==1?jl(g,M,Sf,Tf):jl(g,M,{},bf),T.splice(M,0,g),ya(T[M],M)}n.addSeries=Ea;function ba(g){if(T.splice(g,1),Ze){re.values.splice(g,1),ze.splice(g,1);let M=Ue.splice(g,1)[0];wt(null,M.firstChild),M.remove()}Ke&&(fe.splice(g,1),zn.splice(g,1)[0].remove(),ji.splice(g,1),Bi.splice(g,1)),dn("delSeries",g)}n.delSeries=ba;const Rn=[!1,!1,!1,!1];function ks(g,M){if(g._show=g.show,g.show){let R=g.side%2,U=S[g.scale];U==null&&(g.scale=R?T[1].scale:y,U=S[g.scale]);let V=U.time;g.size=pt(g.size),g.space=pt(g.space),g.rotate=pt(g.rotate),fr(g.incrs)&&g.incrs.forEach(ce=>{!_r.has(ce)&&_r.set(ce,jh(ce))}),g.incrs=pt(g.incrs||(U.distr==2?Xm:V?p==1?qm:Zm:Dr)),g.splits=pt(g.splits||(V&&U.distr==1?j:U.distr==3?Kl:U.distr==4?pg:dg)),g.stroke=pt(g.stroke),g.grid.stroke=pt(g.grid.stroke),g.ticks.stroke=pt(g.ticks.stroke),g.border.stroke=pt(g.border.stroke);let K=g.values;g.values=fr(K)&&!fr(K[0])?pt(K):V?fr(K)?gf(N,mf(K,B)):ff(K)?Qm(N,K):K||ne:K||hg,g.filter=pt(g.filter||(U.distr>=3&&U.log==10?_g:U.distr==3&&U.log==2?xg:Zh)),g.font=Df(g.font),g.labelFont=Df(g.labelFont),g._size=g.size(n,null,M,0),g._space=g._rotate=g._incrs=g._found=g._splits=g._values=null,g._size>0&&(Rn[M]=!0,g._el=ri(tm,u))}}function Qi(g,M,R,U){let[V,K,ce,me]=R,ve=M%2,Pe=0;return ve==0&&(me||K)&&(Pe=M==0&&!V||M==2&&!ce?an(vf.size/3):0),ve==1&&(V||ce)&&(Pe=M==1&&!K||M==3&&!me?an(yf.size/2):0),Pe}const Vs=n.padding=(i.padding||[Qi,Qi,Qi,Qi]).map(g=>pt(vt(g,Qi))),Gn=n._padding=Vs.map((g,M)=>g(n,M,Rn,0));let Qt,Yt=null,jt=null;const Mr=r==1?T[0].idxs:null;let Hn=null,yr=!1;function Ta(g,M){if(e=g??[],n.data=n._data=e,r==2){Qt=0;for(let R=1;R<T.length;R++)Qt+=e[R][0].length}else{e.length==0&&(n.data=n._data=e=[[]]),Hn=e[0],Qt=Hn.length;let R=e;if(z==2){R=e.slice();let U=R[0]=Array(Qt);for(let V=0;V<Qt;V++)U[V]=V}n._data=e=R}if(Dn(!0),dn("setData"),z==2&&(Se=!0),M!==!1){let R=D;R.auto(n,yr)?Gs():nr(y,R.min,R.max),ke=ke||le.left>=0,qe=!0,Yr()}}n.setData=Ta;function Gs(){yr=!0;let g,M;r==1&&(Qt>0?(Yt=Mr[0]=0,jt=Mr[1]=Qt-1,g=e[0][Yt],M=e[0][jt],z==2?(g=Yt,M=jt):g==M&&(z==3?[g,M]=Po(g,g,D.log,!1):z==4?[g,M]=qc(g,g,D.log,!1):D.time?M=g+an(86400/p):[g,M]=_o(g,M,Kc,!0))):(Yt=Mr[0]=g=null,jt=Mr[1]=M=null)),nr(y,g,M)}let b,W,J,q,Z,Te,De,be,Oe,Ce;function et(g,M,R,U,V,K){g??(g=qu),R??(R=Zc),U??(U="butt"),V??(V=qu),K??(K="round"),g!=b&&(h.strokeStyle=b=g),V!=W&&(h.fillStyle=W=V),M!=J&&(h.lineWidth=J=M),K!=Z&&(h.lineJoin=Z=K),U!=Te&&(h.lineCap=Te=U),R!=q&&h.setLineDash(q=R)}function it(g,M,R,U){M!=W&&(h.fillStyle=W=M),g!=De&&(h.font=De=g),R!=be&&(h.textAlign=be=R),U!=Oe&&(h.textBaseline=Oe=U)}function He(g,M,R,U,V=0){if(U.length>0&&g.auto(n,yr)&&(M==null||M.min==null)){let K=vt(Yt,0),ce=vt(jt,U.length-1),me=R.min==null?_m(U,K,ce,V,g.distr==3):[R.min,R.max];g.min=_i(g.min,R.min=me[0]),g.max=On(g.max,R.max=me[1])}}const Mt={min:null,max:null};function qt(){for(let U in S){let V=S[U];L[U]==null&&(V.min==null||L[y]!=null&&V.auto(n,yr))&&(L[U]=Mt)}for(let U in S){let V=S[U];L[U]==null&&V.from!=null&&L[V.from]!=null&&(L[U]=Mt)}L[y]!=null&&Dn(!0);let g={};for(let U in L){let V=L[U];if(V!=null){let K=g[U]=Cs(S[U],Am);if(V.min!=null)nn(K,V);else if(U!=y||r==2)if(Qt==0&&K.from==null){let ce=K.range(n,null,null,U);K.min=ce[0],K.max=ce[1]}else K.min=Ut,K.max=-Ut}}if(Qt>0){T.forEach((U,V)=>{if(r==1){let K=U.scale,ce=L[K];if(ce==null)return;let me=g[K];if(V==0){let ve=me.range(n,me.min,me.max,K);me.min=ve[0],me.max=ve[1],Yt=gi(me.min,e[0]),jt=gi(me.max,e[0]),jt-Yt>1&&(e[0][Yt]<me.min&&Yt++,e[0][jt]>me.max&&jt--),U.min=Hn[Yt],U.max=Hn[jt]}else U.show&&U.auto&&He(me,ce,U,e[V],U.sorted);U.idxs[0]=Yt,U.idxs[1]=jt}else if(V>0&&U.show&&U.auto){let[K,ce]=U.facets,me=K.scale,ve=ce.scale,[Pe,Ie]=e[V],Fe=g[me],tt=g[ve];Fe!=null&&He(Fe,L[me],K,Pe,K.sorted),tt!=null&&He(tt,L[ve],ce,Ie,ce.sorted),U.min=ce.min,U.max=ce.max}});for(let U in g){let V=g[U],K=L[U];if(V.from==null&&(K==null||K.min==null)){let ce=V.range(n,V.min==Ut?null:V.min,V.max==-Ut?null:V.max,U);V.min=ce[0],V.max=ce[1]}}}for(let U in g){let V=g[U];if(V.from!=null){let K=g[V.from];if(K.min==null)V.min=V.max=null;else{let ce=V.range(n,K.min,K.max,U);V.min=ce[0],V.max=ce[1]}}}let M={},R=!1;for(let U in g){let V=g[U],K=S[U];if(K.min!=V.min||K.max!=V.max){K.min=V.min,K.max=V.max;let ce=K.distr;K._min=ce==3?Yi(K.min):ce==4?tl(K.min,K.asinh):ce==100?K.fwd(K.min):K.min,K._max=ce==3?Yi(K.max):ce==4?tl(K.max,K.asinh):ce==100?K.fwd(K.max):K.max,M[U]=R=!0}}if(R){T.forEach((U,V)=>{r==2?V>0&&M.y&&(U._paths=null):M[U.scale]&&(U._paths=null)});for(let U in M)Se=!0,dn("setScale",U);Ke&&le.left>=0&&(ke=qe=!0)}for(let U in L)L[U]=null}function Ht(g){let M=ql(Yt-1,0,Qt-1),R=ql(jt+1,0,Qt-1);for(;g[M]==null&&M>0;)M--;for(;g[R]==null&&R<Qt-1;)R++;return[M,R]}function Rt(){if(Qt>0){let g=T.some(M=>M._focus)&&Ce!=lt.alpha;g&&(h.globalAlpha=Ce=lt.alpha),T.forEach((M,R)=>{if(R>0&&M.show&&(rn(R,!1),rn(R,!0),M._paths==null)){let U=Ce;Ce!=M.alpha&&(h.globalAlpha=Ce=M.alpha);let V=r==2?[0,e[R][0].length-1]:Ht(e[R]);M._paths=M.paths(n,R,V[0],V[1]),Ce!=U&&(h.globalAlpha=Ce=U)}}),T.forEach((M,R)=>{if(R>0&&M.show){let U=Ce;Ce!=M.alpha&&(h.globalAlpha=Ce=M.alpha),M._paths!=null&&Le(R,!1);{let V=M._paths!=null?M._paths.gaps:null,K=M.points.show(n,R,Yt,jt,V),ce=M.points.filter(n,R,K,V);(K||ce)&&(M.points._paths=M.points.paths(n,R,Yt,jt,ce),Le(R,!0))}Ce!=U&&(h.globalAlpha=Ce=U),dn("drawSeries",R)}}),g&&(h.globalAlpha=Ce=1)}}function rn(g,M){let R=M?T[g].points:T[g];R._stroke=R.stroke(n,g),R._fill=R.fill(n,g)}function Le(g,M){let R=M?T[g].points:T[g],{stroke:U,fill:V,clip:K,flags:ce,_stroke:me=R._stroke,_fill:ve=R._fill,_width:Pe=R.width}=R._paths;Pe=Nt(Pe*bt,3);let Ie=null,Fe=Pe%2/2;M&&ve==null&&(ve=Pe>0?"#fff":me);let tt=R.pxAlign==1&&Fe>0;if(tt&&h.translate(Fe,Fe),!M){let Et=_e-Pe/2,xt=te-Pe/2,ct=ie+Pe,Qe=xe+Pe;Ie=new Path2D,Ie.rect(Et,xt,ct,Qe)}M?En(me,Pe,R.dash,R.cap,ve,U,V,ce,K):Cn(g,me,Pe,R.dash,R.cap,ve,U,V,ce,Ie,K),tt&&h.translate(-Fe,-Fe)}function Cn(g,M,R,U,V,K,ce,me,ve,Pe,Ie){let Fe=!1;ve!=0&&C.forEach((tt,Et)=>{if(tt.series[0]==g){let xt=T[tt.series[1]],ct=e[tt.series[1]],Qe=(xt._paths||ma).band;fr(Qe)&&(Qe=tt.dir==1?Qe[0]:Qe[1]);let We,kt=null;xt.show&&Qe&&vm(ct,Yt,jt)?(kt=tt.fill(n,Et)||K,We=xt._paths.clip):Qe=null,En(M,R,U,V,kt,ce,me,ve,Pe,Ie,We,Qe),Fe=!0}}),Fe||En(M,R,U,V,K,ce,me,ve,Pe,Ie)}const dt=Ps|Zl;function En(g,M,R,U,V,K,ce,me,ve,Pe,Ie,Fe){et(g,M,R,U,V),(ve||Pe||Fe)&&(h.save(),ve&&h.clip(ve),Pe&&h.clip(Pe)),Fe?(me&dt)==dt?(h.clip(Fe),Ie&&h.clip(Ie),Wn(V,ce),Pn(g,K,M)):me&Zl?(Wn(V,ce),h.clip(Fe),Pn(g,K,M)):me&Ps&&(h.save(),h.clip(Fe),Ie&&h.clip(Ie),Wn(V,ce),h.restore(),Pn(g,K,M)):(Wn(V,ce),Pn(g,K,M)),(ve||Pe||Fe)&&h.restore()}function Pn(g,M,R){R>0&&(M instanceof Map?M.forEach((U,V)=>{h.strokeStyle=b=V,h.stroke(U)}):M!=null&&g&&h.stroke(M))}function Wn(g,M){M instanceof Map?M.forEach((R,U)=>{h.fillStyle=W=U,h.fill(R)}):M!=null&&g&&h.fill(M)}function er(g,M,R,U){let V=P[g],K;if(U<=0)K=[0,0];else{let ce=V._space=V.space(n,g,M,R,U),me=V._incrs=V.incrs(n,g,M,R,U,ce);K=kg(M,R,me,U,ce)}return V._found=K}function yt(g,M,R,U,V,K,ce,me,ve,Pe){let Ie=ce%2/2;E==1&&h.translate(Ie,Ie),et(me,ce,ve,Pe,me),h.beginPath();let Fe,tt,Et,xt,ct=V+(U==0||U==3?-K:K);R==0?(tt=V,xt=ct):(Fe=V,Et=ct);for(let Qe=0;Qe<g.length;Qe++)M[Qe]!=null&&(R==0?Fe=Et=g[Qe]:tt=xt=g[Qe],h.moveTo(Fe,tt),h.lineTo(Et,xt));h.stroke(),E==1&&h.translate(-Ie,-Ie)}function Kt(g){let M=!0;return P.forEach((R,U)=>{if(!R.show)return;let V=S[R.scale];if(V.min==null){R._show&&(M=!1,R._show=!1,Dn(!1));return}else R._show||(M=!1,R._show=!0,Dn(!1));let K=R.side,ce=K%2,{min:me,max:ve}=V,[Pe,Ie]=er(U,me,ve,ce==0?Xe:pe);if(Ie==0)return;let Fe=V.distr==2,tt=R._splits=R.splits(n,U,me,ve,Pe,Ie,Fe),Et=V.distr==2?tt.map(We=>Hn[We]):tt,xt=V.distr==2?Hn[tt[1]]-Hn[tt[0]]:Pe,ct=R._values=R.values(n,R.filter(n,Et,U,Ie,xt),U,Ie,xt);R._rotate=K==2?R.rotate(n,ct,U,Ie):0;let Qe=R._size;R._size=si(R.size(n,ct,U,g)),Qe!=null&&R._size!=Qe&&(M=!1)}),M}function Si(g){let M=!0;return Vs.forEach((R,U)=>{let V=R(n,U,Rn,g);V!=Gn[U]&&(M=!1),Gn[U]=V}),M}function Pt(){for(let g=0;g<P.length;g++){let M=P[g];if(!M.show||!M._show)continue;let R=M.side,U=R%2,V,K,ce=M.stroke(n,g),me=R==0||R==3?-1:1,[ve,Pe]=M._found;if(M.label!=null){let In=M.labelGap*me,qn=an((M._lpos+In)*bt);it(M.labelFont[0],ce,"center",R==2?Zs:Yu),h.save(),U==1?(V=K=0,h.translate(qn,an(te+xe/2)),h.rotate((R==3?-ao:ao)/2)):(V=an(_e+ie/2),K=qn);let Tr=Kh(M.label)?M.label(n,g,ve,Pe):M.label;h.fillText(Tr,V,K),h.restore()}if(Pe==0)continue;let Ie=S[M.scale],Fe=U==0?ie:xe,tt=U==0?_e:te,Et=M._splits,xt=Ie.distr==2?Et.map(In=>Hn[In]):Et,ct=Ie.distr==2?Hn[Et[1]]-Hn[Et[0]]:ve,Qe=M.ticks,We=M.border,kt=Qe.show?Qe.size:0,Zt=an(kt*bt),pn=an((M.alignTo==2?M._size-kt-M.gap:M.gap)*bt),Dt=M._rotate*-ao/180,Jt=m(M._pos*bt),Xn=(Zt+pn)*me,Ln=Jt+Xn;K=U==0?Ln:0,V=U==1?Ln:0;let ti=M.font[0],ui=M.align==1?ts:M.align==2?jo:Dt>0?ts:Dt<0?jo:U==0?"center":R==3?jo:ts,Ei=Dt||U==1?"middle":R==2?Zs:Yu;it(ti,ce,ui,Ei);let Yn=M.font[1]*M.lineGap,ni=Et.map(In=>m(o(In,Ie,Fe,tt))),fi=M._values;for(let In=0;In<fi.length;In++){let qn=fi[In];if(qn!=null){U==0?V=ni[In]:K=ni[In],qn=""+qn;let Tr=qn.indexOf(`
`)==-1?[qn]:qn.split(/\n/gm);for(let Un=0;Un<Tr.length;Un++){let Gu=Tr[Un];Dt?(h.save(),h.translate(V,K+Un*Yn),h.rotate(Dt),h.fillText(Gu,0,0),h.restore()):h.fillText(Gu,V,K+Un*Yn)}}}Qe.show&&yt(ni,Qe.filter(n,xt,g,Pe,ct),U,R,Jt,Zt,Nt(Qe.width*bt,3),Qe.stroke(n,g),Qe.dash,Qe.cap);let bi=M.grid;bi.show&&yt(ni,bi.filter(n,xt,g,Pe,ct),U,U==0?2:1,U==0?te:_e,U==0?xe:ie,Nt(bi.width*bt,3),bi.stroke(n,g),bi.dash,bi.cap),We.show&&yt([Jt],[1],U==0?1:0,U==0?1:2,U==1?te:_e,U==1?xe:ie,Nt(We.width*bt,3),We.stroke(n,g),We.dash,We.cap)}dn("drawAxes")}function Dn(g){T.forEach((M,R)=>{R>0&&(M._paths=null,g&&(r==1?(M.min=null,M.max=null):M.facets.forEach(U=>{U.min=null,U.max=null})))})}let ci=!1,Xr=!1,Hs=[];function mp(){Xr=!1;for(let g=0;g<Hs.length;g++)dn(...Hs[g]);Hs.length=0}function Yr(){ci||(Im(bu),ci=!0)}function gp(g,M=!1){ci=!0,Xr=M,g(n),bu(),M&&Hs.length>0&&queueMicrotask(mp)}n.batch=gp;function bu(){if(Ve&&(qt(),Ve=!1),Se&&(Ee(),Se=!1),ye){if(Vt(d,ts,w),Vt(d,Zs,_),Vt(d,la,Xe),Vt(d,ca,pe),Vt(x,ts,w),Vt(x,Zs,_),Vt(x,la,Xe),Vt(x,ca,pe),Vt(u,la,Tt),Vt(u,ca,H),f.width=an(Tt*bt),f.height=an(H*bt),P.forEach(({_el:g,_show:M,_size:R,_pos:U,side:V})=>{if(g!=null)if(M){let K=V===3||V===0?R:0,ce=V%2==1;Vt(g,ce?"left":"top",U-K),Vt(g,ce?"width":"height",R),Vt(g,ce?"top":"left",ce?_:w),Vt(g,ce?"height":"width",ce?pe:Xe),Xl(g,Lr)}else Zn(g,Lr)}),b=W=J=Z=Te=De=be=Oe=q=null,Ce=1,Ys(!0),w!=Y||_!=$||Xe!=ee||pe!=ge){Dn(!1);let g=Xe/ee,M=pe/ge;if(Ke&&!ke&&le.left>=0){le.left*=g,le.top*=M,qr&&Ti(qr,an(le.left),0,Xe,pe),Kr&&Ti(Kr,0,an(le.top),Xe,pe);for(let R=0;R<zn.length;R++){let U=zn[R];U!=null&&(ji[R]*=g,Bi[R]*=M,Ti(U,si(ji[R]),si(Bi[R]),Xe,pe))}}if(zt.show&&!Ye&&zt.left>=0&&zt.width>0){zt.left*=g,zt.width*=g,zt.top*=M,zt.height*=M;for(let R in Ko)Vt(Jr,R,zt[R])}Y=w,$=_,ee=Xe,ge=pe}dn("setSize"),ye=!1}Tt>0&&H>0&&(h.clearRect(0,0,f.width,f.height),dn("drawClear"),v.forEach(g=>g()),dn("draw")),zt.show&&Ye&&(Aa(zt),Ye=!1),Ke&&ke&&(br(null,!0,!1),ke=!1),re.show&&re.live&&qe&&(Yo(),qe=!1),l||(l=!0,n.status=1,dn("ready")),yr=!1,ci=!1}n.redraw=(g,M)=>{Se=M||!1,g!==!1?nr(y,D.min,D.max):Yr()};function Ho(g,M){let R=S[g];if(R.from==null){if(Qt==0){let U=R.range(n,M.min,M.max,g);M.min=U[0],M.max=U[1]}if(M.min>M.max){let U=M.min;M.min=M.max,M.max=U}if(Qt>1&&M.min!=null&&M.max!=null&&M.max-M.min<1e-16)return;g==y&&R.distr==2&&Qt>0&&(M.min=gi(M.min,e[0]),M.max=gi(M.max,e[0]),M.min==M.max&&M.max++),L[g]=M,Ve=!0,Yr()}}n.setScale=Ho;let Wo,Xo,qr,Kr,Tu,Au,$r,Zr,wu,Ru,Bt,Wt,tr=!1;const Sn=le.drag;let fn=Sn.x,hn=Sn.y;Ke&&(le.x&&(Wo=ri(im,x)),le.y&&(Xo=ri(rm,x)),D.ori==0?(qr=Wo,Kr=Xo):(qr=Xo,Kr=Wo),Bt=le.left,Wt=le.top);const zt=n.select=nn({show:!0,over:!0,left:0,width:0,top:0,height:0},i.select),Jr=zt.show?ri(nm,zt.over?x:d):null;function Aa(g,M){if(zt.show){for(let R in g)zt[R]=g[R],R in Ko&&Vt(Jr,R,g[R]);M!==!1&&dn("setSelect")}}n.setSelect=Aa;function _p(g){if(T[g].show)Ze&&Xl(Ue[g],Lr);else if(Ze&&Zn(Ue[g],Lr),Ke){let R=vn?zn[0]:zn[g];R!=null&&Ti(R,-10,-10,Xe,pe)}}function nr(g,M,R){Ho(g,{min:M,max:R})}function Mi(g,M,R,U){M.focus!=null&&yp(g),M.show!=null&&T.forEach((V,K)=>{K>0&&(g==K||g==null)&&(V.show=M.show,_p(K),r==2?(nr(V.facets[0].scale,null,null),nr(V.facets[1].scale,null,null)):nr(V.scale,null,null),Yr())}),R!==!1&&dn("setSeries",g,M),U&&qs("setSeries",n,g,M)}n.setSeries=Mi;function xp(g,M){nn(C[g],M)}function vp(g,M){g.fill=pt(g.fill||null),g.dir=vt(g.dir,-1),M=M??C.length,C.splice(M,0,g)}function Sp(g){g==null?C.length=0:C.splice(g,1)}n.addBand=vp,n.setBand=xp,n.delBand=Sp;function Mp(g,M){T[g].alpha=M,Ke&&zn[g]!=null&&(zn[g].style.opacity=M),Ze&&Ue[g]&&(Ue[g].style.opacity=M)}let zi,ir,Er;const jr={focus:!0};function yp(g){if(g!=Er){let M=g==null,R=lt.alpha!=1;T.forEach((U,V)=>{if(r==1||V>0){let K=M||V==0||V==g;U._focus=M?null:K,R&&Mp(V,K?1:lt.alpha)}}),Er=g,R&&Yr()}}Ze&&xn&&_t(Zu,ue,g=>{le._lock||(Ne(g),Er!=null&&Mi(null,jr,!0,en.setSeries))});function yi(g,M,R){let U=S[M];R&&(g=g/bt-(U.ori==1?_:w));let V=Xe;U.ori==1&&(V=pe,g=V-g),U.dir==-1&&(g=V-g);let K=U._min,ce=U._max,me=g/V,ve=K+(ce-K)*me,Pe=U.distr;return Pe==3?ws(10,ve):Pe==4?Mm(ve,U.asinh):Pe==100?U.bwd(ve):ve}function Ep(g,M){let R=yi(g,y,M);return gi(R,e[0],Yt,jt)}n.valToIdx=g=>gi(g,e[0]),n.posToIdx=Ep,n.posToVal=yi,n.valToPos=(g,M,R)=>S[M].ori==0?s(g,S[M],R?ie:Xe,R?_e:0):a(g,S[M],R?xe:pe,R?te:0),n.setCursor=(g,M,R)=>{Bt=g.left,Wt=g.top,br(null,M,R)};function Cu(g,M){Vt(Jr,ts,zt.left=g),Vt(Jr,la,zt.width=M)}function Pu(g,M){Vt(Jr,Zs,zt.top=g),Vt(Jr,ca,zt.height=M)}let Ws=D.ori==0?Cu:Pu,Xs=D.ori==1?Cu:Pu;function bp(){if(Ze&&re.live)for(let g=r==2?1:0;g<T.length;g++){if(g==0&&je)continue;let M=re.values[g],R=0;for(let U in M)ze[g][R++].firstChild.nodeValue=M[U]}}function Yo(g,M){if(g!=null&&(g.idxs?g.idxs.forEach((R,U)=>{fe[U]=R}):Tm(g.idx)||fe.fill(g.idx),re.idx=fe[0]),Ze&&re.live){for(let R=0;R<T.length;R++)(R>0||r==1&&!je)&&Tp(R,fe[R]);bp()}qe=!1,M!==!1&&dn("setLegend")}n.setLegend=Yo;function Tp(g,M){let R=T[g],U=g==0&&z==2?Hn:e[g],V;je?V=R.values(n,g,M)??ht:(V=R.value(n,M==null?null:U[M],g,M),V=V==null?ht:{_:V}),re.values[g]=V}function br(g,M,R){wu=Bt,Ru=Wt,[Bt,Wt]=le.move(n,Bt,Wt),le.left=Bt,le.top=Wt,Ke&&(qr&&Ti(qr,an(Bt),0,Xe,pe),Kr&&Ti(Kr,0,an(Wt),Xe,pe));let U,V=Yt>jt;zi=Ut,ir=null;let K=D.ori==0?Xe:pe,ce=D.ori==1?Xe:pe;if(Bt<0||Qt==0||V){U=le.idx=null;for(let me=0;me<T.length;me++){let ve=zn[me];ve!=null&&Ti(ve,-10,-10,Xe,pe)}xn&&Mi(null,jr,!0,g==null&&en.setSeries),re.live&&(fe.fill(U),qe=!0)}else{let me,ve,Pe;r==1&&(me=D.ori==0?Bt:Wt,ve=yi(me,y),U=le.idx=gi(ve,e[0],Yt,jt),Pe=O(e[0][U],D,K,0));let Ie=-10,Fe=-10,tt=0,Et=0,xt=!0,ct="",Qe="";for(let We=r==2?1:0;We<T.length;We++){let kt=T[We],Zt=fe[We],pn=Zt==null?null:r==1?e[We][Zt]:e[We][1][Zt],Dt=le.dataIdx(n,We,U,ve),Jt=Dt==null?null:r==1?e[We][Dt]:e[We][1][Dt];if(qe=qe||Jt!=pn||Dt!=Zt,fe[We]=Dt,We>0&&kt.show){let Xn=Dt==null?-10:Dt==U?Pe:O(r==1?e[0][Dt]:e[We][0][Dt],D,K,0),Ln=Jt==null?-10:k(Jt,r==1?S[kt.scale]:S[kt.facets[1].scale],ce,0);if(xn&&Jt!=null){let ti=D.ori==1?Bt:Wt,ui=on(lt.dist(n,We,Dt,Ln,ti));if(ui<zi){let Ei=lt.bias;if(Ei!=0){let Yn=yi(ti,kt.scale),ni=Jt>=0?1:-1,fi=Yn>=0?1:-1;fi==ni&&(fi==1?Ei==1?Jt>=Yn:Jt<=Yn:Ei==1?Jt<=Yn:Jt>=Yn)&&(zi=ui,ir=We)}else zi=ui,ir=We}}if(qe||vn){let ti,ui;D.ori==0?(ti=Xn,ui=Ln):(ti=Ln,ui=Xn);let Ei,Yn,ni,fi,bi,In,qn=!0,Tr=at.bbox;if(Tr!=null){qn=!1;let Un=Tr(n,We);ni=Un.left,fi=Un.top,Ei=Un.width,Yn=Un.height}else ni=ti,fi=ui,Ei=Yn=at.size(n,We);if(In=at.fill(n,We),bi=at.stroke(n,We),vn)We==ir&&zi<=lt.prox&&(Ie=ni,Fe=fi,tt=Ei,Et=Yn,xt=qn,ct=In,Qe=bi);else{let Un=zn[We];Un!=null&&(ji[We]=ni,Bi[We]=fi,rf(Un,Ei,Yn,qn),tf(Un,In,bi),Ti(Un,si(ni),si(fi),Xe,pe))}}}}if(vn){let We=lt.prox,kt=Er==null?zi<=We:zi>We||ir!=Er;if(qe||kt){let Zt=zn[0];Zt!=null&&(ji[0]=Ie,Bi[0]=Fe,rf(Zt,tt,Et,xt),tf(Zt,ct,Qe),Ti(Zt,si(Ie),si(Fe),Xe,pe))}}}if(zt.show&&tr)if(g!=null){let[me,ve]=en.scales,[Pe,Ie]=en.match,[Fe,tt]=g.cursor.sync.scales,Et=g.cursor.drag;if(fn=Et._x,hn=Et._y,fn||hn){let{left:xt,top:ct,width:Qe,height:We}=g.select,kt=g.scales[Fe].ori,Zt=g.posToVal,pn,Dt,Jt,Xn,Ln,ti=me!=null&&Pe(me,Fe),ui=ve!=null&&Ie(ve,tt);ti&&fn?(kt==0?(pn=xt,Dt=Qe):(pn=ct,Dt=We),Jt=S[me],Xn=O(Zt(pn,Fe),Jt,K,0),Ln=O(Zt(pn+Dt,Fe),Jt,K,0),Ws(_i(Xn,Ln),on(Ln-Xn))):Ws(0,K),ui&&hn?(kt==1?(pn=xt,Dt=Qe):(pn=ct,Dt=We),Jt=S[ve],Xn=k(Zt(pn,tt),Jt,ce,0),Ln=k(Zt(pn+Dt,tt),Jt,ce,0),Xs(_i(Xn,Ln),on(Ln-Xn))):Xs(0,ce)}else $o()}else{let me=on(wu-Tu),ve=on(Ru-Au);if(D.ori==1){let tt=me;me=ve,ve=tt}fn=Sn.x&&me>=Sn.dist,hn=Sn.y&&ve>=Sn.dist;let Pe=Sn.uni;Pe!=null?fn&&hn&&(fn=me>=Pe,hn=ve>=Pe,!fn&&!hn&&(ve>me?hn=!0:fn=!0)):Sn.x&&Sn.y&&(fn||hn)&&(fn=hn=!0);let Ie,Fe;fn&&(D.ori==0?(Ie=$r,Fe=Bt):(Ie=Zr,Fe=Wt),Ws(_i(Ie,Fe),on(Fe-Ie)),hn||Xs(0,ce)),hn&&(D.ori==1?(Ie=$r,Fe=Bt):(Ie=Zr,Fe=Wt),Xs(_i(Ie,Fe),on(Fe-Ie)),fn||Ws(0,K)),!fn&&!hn&&(Ws(0,0),Xs(0,0))}if(Sn._x=fn,Sn._y=hn,g==null){if(R){if(Vu!=null){let[me,ve]=en.scales;en.values[0]=me!=null?yi(D.ori==0?Bt:Wt,me):null,en.values[1]=ve!=null?yi(D.ori==1?Bt:Wt,ve):null}qs(Qo,n,Bt,Wt,Xe,pe,U)}if(xn){let me=R&&en.setSeries,ve=lt.prox;Er==null?zi<=ve&&Mi(ir,jr,!0,me):zi>ve?Mi(null,jr,!0,me):ir!=Er&&Mi(ir,jr,!0,me)}}qe&&(re.idx=U,Yo()),M!==!1&&dn("setCursor")}let rr=null;Object.defineProperty(n,"rect",{get(){return rr==null&&Ys(!1),rr}});function Ys(g=!1){g?rr=null:(rr=x.getBoundingClientRect(),dn("syncRect",rr))}function Du(g,M,R,U,V,K,ce){le._lock||tr&&g!=null&&g.movementX==0&&g.movementY==0||(qo(g,M,R,U,V,K,ce,!1,g!=null),g!=null?br(null,!0,!0):br(M,!0,!1))}function qo(g,M,R,U,V,K,ce,me,ve){if(rr==null&&Ys(!1),Ne(g),g!=null)R=g.clientX-rr.left,U=g.clientY-rr.top;else{if(R<0||U<0){Bt=-10,Wt=-10;return}let[Pe,Ie]=en.scales,Fe=M.cursor.sync,[tt,Et]=Fe.values,[xt,ct]=Fe.scales,[Qe,We]=en.match,kt=M.axes[0].side%2==1,Zt=D.ori==0?Xe:pe,pn=D.ori==1?Xe:pe,Dt=kt?K:V,Jt=kt?V:K,Xn=kt?U:R,Ln=kt?R:U;if(xt!=null?R=Qe(Pe,xt)?o(tt,S[Pe],Zt,0):-10:R=Zt*(Xn/Dt),ct!=null?U=We(Ie,ct)?o(Et,S[Ie],pn,0):-10:U=pn*(Ln/Jt),D.ori==1){let ti=R;R=U,U=ti}}ve&&(M==null||M.cursor.event.type==Qo)&&((R<=1||R>=Xe-1)&&(R=Pr(R,Xe)),(U<=1||U>=pe-1)&&(U=Pr(U,pe))),me?(Tu=R,Au=U,[$r,Zr]=le.move(n,R,U)):(Bt=R,Wt=U)}const Ko={width:0,height:0,left:0,top:0};function $o(){Aa(Ko,!1)}let Lu,Iu,Uu,Nu;function Fu(g,M,R,U,V,K,ce){tr=!0,fn=hn=Sn._x=Sn._y=!1,qo(g,M,R,U,V,K,ce,!0,!1),g!=null&&(_t(el,Hl,Ou,!1),qs(Ku,n,$r,Zr,Xe,pe,null));let{left:me,top:ve,width:Pe,height:Ie}=zt;Lu=me,Iu=ve,Uu=Pe,Nu=Ie}function Ou(g,M,R,U,V,K,ce){tr=Sn._x=Sn._y=!1,qo(g,M,R,U,V,K,ce,!1,!0);let{left:me,top:ve,width:Pe,height:Ie}=zt,Fe=Pe>0||Ie>0,tt=Lu!=me||Iu!=ve||Uu!=Pe||Nu!=Ie;if(Fe&&tt&&Aa(zt),Sn.setScale&&Fe&&tt){let Et=me,xt=Pe,ct=ve,Qe=Ie;if(D.ori==1&&(Et=ve,xt=Ie,ct=me,Qe=Pe),fn&&nr(y,yi(Et,y),yi(Et+xt,y)),hn)for(let We in S){let kt=S[We];We!=y&&kt.from==null&&kt.min!=Ut&&nr(We,yi(ct+Qe,We),yi(ct,We))}$o()}else le.lock&&(le._lock=!le._lock,br(M,!0,g!=null));g!=null&&(wt(el,Hl),qs(el,n,Bt,Wt,Xe,pe,null))}function Ap(g,M,R,U,V,K,ce){if(le._lock)return;Ne(g);let me=tr;if(tr){let ve=!0,Pe=!0,Ie=10,Fe,tt;D.ori==0?(Fe=fn,tt=hn):(Fe=hn,tt=fn),Fe&&tt&&(ve=Bt<=Ie||Bt>=Xe-Ie,Pe=Wt<=Ie||Wt>=pe-Ie),Fe&&ve&&(Bt=Bt<$r?0:Xe),tt&&Pe&&(Wt=Wt<Zr?0:pe),br(null,!0,!0),tr=!1}Bt=-10,Wt=-10,fe.fill(null),br(null,!0,!0),me&&(tr=me)}function Bu(g,M,R,U,V,K,ce){le._lock||(Ne(g),Gs(),$o(),g!=null&&qs(Ju,n,Bt,Wt,Xe,pe,null))}function zu(){P.forEach(Vg),G(n.width,n.height,!0)}Br(go,ys,zu);const Qr={};Qr.mousedown=Fu,Qr.mousemove=Du,Qr.mouseup=Ou,Qr.dblclick=Bu,Qr.setSeries=(g,M,R,U)=>{let V=en.match[2];R=V(n,M,R),R!=-1&&Mi(R,U,!0,!1)},Ke&&(_t(Ku,x,Fu),_t(Qo,x,Du),_t($u,x,g=>{Ne(g),Ys(!1)}),_t(Zu,x,Ap),_t(Ju,x,Bu),Jl.add(n),n.syncRect=Ys);const wa=n.hooks=i.hooks||{};function dn(g,M,R){Xr?Hs.push([g,M,R]):g in wa&&wa[g].forEach(U=>{U.call(null,n,M,R)})}(i.plugins||[]).forEach(g=>{for(let M in g.hooks)wa[M]=(wa[M]||[]).concat(g.hooks[M])});const ku=(g,M,R)=>R,en=nn({key:null,setSeries:!1,filters:{pub:cf,sub:cf},scales:[y,T[1]?T[1].scale:null],match:[uf,uf,ku],values:[null,null]},le.sync);en.match.length==2&&en.match.push(ku),le.sync=en;const Vu=en.key,Zo=xd(Vu);function qs(g,M,R,U,V,K,ce){en.filters.pub(g,M,R,U,V,K,ce)&&Zo.pub(g,M,R,U,V,K,ce)}Zo.sub(n);function wp(g,M,R,U,V,K,ce){en.filters.sub(g,M,R,U,V,K,ce)&&Qr[g](null,M,R,U,V,K,ce)}n.pub=wp;function Rp(){Zo.unsub(n),Jl.delete(n),Ot.clear(),Yl(go,ys,zu),c.remove(),ue==null||ue.remove(),dn("destroy")}n.destroy=Rp;function Jo(){dn("init",i,e),Ta(e||i.data,!1),L[y]?Ho(y,L[y]):Gs(),Ye=zt.show&&(zt.width>0||zt.height>0),ke=qe=!0,G(i.width,i.height)}return T.forEach(ya),P.forEach(ks),t?t instanceof HTMLElement?(t.appendChild(c),Jo()):t(n,Jo):Jo(),n}An.assign=nn;An.fmtNum=$c;An.rangeNum=_o;An.rangeLog=Po;An.rangeAsinh=qc;An.orient=Hr;An.pxRatio=bt;An.join=Lm;An.fmtDate=Jc,An.tzDate=Hm;An.sync=xd;{An.addGap=Tg,An.clipGaps=Io;let i=An.paths={points:bd};i.linear=Ad,i.stepped=Rg,i.bars=Cg,i.spline=Dg}const Gg=["#00b4d8","#ff6b6b","#51cf66","#ffd43b"];function Hg({title:i,data:e,series:t,colors:n=Gg,height:r=200}){const s=Ae.useRef(null),a=Ae.useRef(null);return Ae.useEffect(()=>{if(!s.current||e[0].length===0)return;const o={width:s.current.clientWidth||400,height:r,cursor:{show:!0},legend:{show:!0,live:!1},axes:[{stroke:"rgba(255,255,255,0.4)",grid:{stroke:"rgba(255,255,255,0.06)"},ticks:{stroke:"rgba(255,255,255,0.1)"}},{stroke:"rgba(255,255,255,0.4)",grid:{stroke:"rgba(255,255,255,0.06)"},ticks:{stroke:"rgba(255,255,255,0.1)"}}],scales:{x:{time:!0},y:{auto:!0}},series:[{label:"Time",value:(c,f)=>f?new Date(f*1e3).toISOString().slice(11,23):"-"},...t.map((c,f)=>({label:c,stroke:n[f%n.length],width:1.5,points:{show:e[0].length<200}}))]},l=new An(o,e,s.current);return a.current=l,()=>{l.destroy(),a.current=null}},[e,t,n,r]),se.jsxs("div",{style:{background:"var(--gray-900)",borderRadius:"var(--radius-md)",overflow:"hidden"},children:[i&&se.jsx("div",{style:{padding:"6px 12px",fontSize:"var(--font-size-xs)",color:"var(--gray-400)",borderBottom:"1px solid rgba(255,255,255,0.06)"},children:i}),se.jsx("div",{ref:s})]})}function Lf(i,e=.5){const t=[],n=[],r=Date.now()/1e3-i;for(let s=0;s<i;s++)t.push(r+s),n.push(Math.sin(s*e)+Math.random()*.3);return[t,n]}/**
 * @license
 * Copyright 2010-2026 Three.js Authors
 * SPDX-License-Identifier: MIT
 */const ru="185",Es={ROTATE:0,DOLLY:1,PAN:2},Ms={ROTATE:0,PAN:1,DOLLY_PAN:2,DOLLY_ROTATE:3},Wg=0,If=1,Xg=2,oo=1,Yg=2,fa=3,xr=0,Vn=1,Pi=2,Ki=0,bs=1,Uf=2,Nf=3,Ff=4,qg=5,Ir=100,Kg=101,$g=102,Zg=103,Jg=104,jg=200,Qg=201,e0=202,t0=203,Ql=204,ec=205,n0=206,i0=207,r0=208,s0=209,a0=210,o0=211,l0=212,c0=213,u0=214,tc=0,nc=1,ic=2,Ds=3,rc=4,sc=5,ac=6,oc=7,Pd=0,f0=1,h0=2,Ui=0,Dd=1,Ld=2,Id=3,Ud=4,Nd=5,Fd=6,Od=7,Bd=300,kr=301,Ls=302,sl=303,al=304,Oo=306,lc=1e3,Xi=1001,cc=1002,Mn=1003,d0=1004,Da=1005,yn=1006,ol=1007,Fr=1008,Jn=1009,zd=1010,kd=1011,_a=1012,su=1013,Fi=1014,Li=1015,Zi=1016,au=1017,ou=1018,xa=1020,Vd=35902,Gd=35899,Hd=1021,Wd=1022,xi=1023,Ji=1026,Or=1027,Xd=1028,lu=1029,Vr=1030,cu=1031,uu=1033,lo=33776,co=33777,uo=33778,fo=33779,uc=35840,fc=35841,hc=35842,dc=35843,pc=36196,mc=37492,gc=37496,_c=37488,xc=37489,vo=37490,vc=37491,Sc=37808,Mc=37809,yc=37810,Ec=37811,bc=37812,Tc=37813,Ac=37814,wc=37815,Rc=37816,Cc=37817,Pc=37818,Dc=37819,Lc=37820,Ic=37821,Uc=36492,Nc=36494,Fc=36495,Oc=36283,Bc=36284,So=36285,zc=36286,p0=3200,Of=0,m0=1,dr="",ai="srgb",Mo="srgb-linear",yo="linear",Ct="srgb",ns=7680,Bf=519,g0=512,_0=513,x0=514,fu=515,v0=516,S0=517,hu=518,M0=519,kc=35044,zf="300 es",Ii=2e3,va=2001;function y0(i){for(let e=i.length-1;e>=0;--e)if(i[e]>=65535)return!0;return!1}function Eo(i){return document.createElementNS("http://www.w3.org/1999/xhtml",i)}function E0(){const i=Eo("canvas");return i.style.display="block",i}const kf={};function bo(...i){const e="THREE."+i.shift();console.log(e,...i)}function Yd(i){const e=i[0];if(typeof e=="string"&&e.startsWith("TSL:")){const t=i[1];t&&t.isStackTrace?i[0]+=" "+t.getLocation():i[1]='Stack trace not available. Enable "THREE.Node.captureStackTrace" to capture stack traces.'}return i}function Je(...i){i=Yd(i);const e="THREE."+i.shift();{const t=i[0];t&&t.isStackTrace?console.warn(t.getError(e)):console.warn(e,...i)}}function gt(...i){i=Yd(i);const e="THREE."+i.shift();{const t=i[0];t&&t.isStackTrace?console.error(t.getError(e)):console.error(e,...i)}}function Ts(...i){const e=i.join(" ");e in kf||(kf[e]=!0,Je(...i))}function b0(i,e,t){return new Promise(function(n,r){function s(){switch(i.clientWaitSync(e,i.SYNC_FLUSH_COMMANDS_BIT,0)){case i.WAIT_FAILED:r();break;case i.TIMEOUT_EXPIRED:setTimeout(s,t);break;default:n()}}setTimeout(s,t)})}const T0={[tc]:nc,[ic]:ac,[rc]:oc,[Ds]:sc,[nc]:tc,[ac]:ic,[oc]:rc,[sc]:Ds};class Sr{addEventListener(e,t){this._listeners===void 0&&(this._listeners={});const n=this._listeners;n[e]===void 0&&(n[e]=[]),n[e].indexOf(t)===-1&&n[e].push(t)}hasEventListener(e,t){const n=this._listeners;return n===void 0?!1:n[e]!==void 0&&n[e].indexOf(t)!==-1}removeEventListener(e,t){const n=this._listeners;if(n===void 0)return;const r=n[e];if(r!==void 0){const s=r.indexOf(t);s!==-1&&r.splice(s,1)}}dispatchEvent(e){const t=this._listeners;if(t===void 0)return;const n=t[e.type];if(n!==void 0){e.target=this;const r=n.slice(0);for(let s=0,a=r.length;s<a;s++)r[s].call(this,e);e.target=null}}}const bn=["00","01","02","03","04","05","06","07","08","09","0a","0b","0c","0d","0e","0f","10","11","12","13","14","15","16","17","18","19","1a","1b","1c","1d","1e","1f","20","21","22","23","24","25","26","27","28","29","2a","2b","2c","2d","2e","2f","30","31","32","33","34","35","36","37","38","39","3a","3b","3c","3d","3e","3f","40","41","42","43","44","45","46","47","48","49","4a","4b","4c","4d","4e","4f","50","51","52","53","54","55","56","57","58","59","5a","5b","5c","5d","5e","5f","60","61","62","63","64","65","66","67","68","69","6a","6b","6c","6d","6e","6f","70","71","72","73","74","75","76","77","78","79","7a","7b","7c","7d","7e","7f","80","81","82","83","84","85","86","87","88","89","8a","8b","8c","8d","8e","8f","90","91","92","93","94","95","96","97","98","99","9a","9b","9c","9d","9e","9f","a0","a1","a2","a3","a4","a5","a6","a7","a8","a9","aa","ab","ac","ad","ae","af","b0","b1","b2","b3","b4","b5","b6","b7","b8","b9","ba","bb","bc","bd","be","bf","c0","c1","c2","c3","c4","c5","c6","c7","c8","c9","ca","cb","cc","cd","ce","cf","d0","d1","d2","d3","d4","d5","d6","d7","d8","d9","da","db","dc","dd","de","df","e0","e1","e2","e3","e4","e5","e6","e7","e8","e9","ea","eb","ec","ed","ee","ef","f0","f1","f2","f3","f4","f5","f6","f7","f8","f9","fa","fb","fc","fd","fe","ff"],ho=Math.PI/180,Vc=180/Math.PI;function gr(){const i=Math.random()*4294967295|0,e=Math.random()*4294967295|0,t=Math.random()*4294967295|0,n=Math.random()*4294967295|0;return(bn[i&255]+bn[i>>8&255]+bn[i>>16&255]+bn[i>>24&255]+"-"+bn[e&255]+bn[e>>8&255]+"-"+bn[e>>16&15|64]+bn[e>>24&255]+"-"+bn[t&63|128]+bn[t>>8&255]+"-"+bn[t>>16&255]+bn[t>>24&255]+bn[n&255]+bn[n>>8&255]+bn[n>>16&255]+bn[n>>24&255]).toLowerCase()}function ut(i,e,t){return Math.max(e,Math.min(t,i))}function A0(i,e){return(i%e+e)%e}function ll(i,e,t){return(1-t)*i+t*e}function Di(i,e){switch(e.constructor){case Float32Array:return i;case Uint32Array:return i/4294967295;case Uint16Array:return i/65535;case Uint8Array:return i/255;case Int32Array:return Math.max(i/2147483647,-1);case Int16Array:return Math.max(i/32767,-1);case Int8Array:return Math.max(i/127,-1);default:throw new Error("THREE.MathUtils: Invalid component type.")}}function Lt(i,e){switch(e.constructor){case Float32Array:return i;case Uint32Array:return Math.round(i*4294967295);case Uint16Array:return Math.round(i*65535);case Uint8Array:return Math.round(i*255);case Int32Array:return Math.round(i*2147483647);case Int16Array:return Math.round(i*32767);case Int8Array:return Math.round(i*127);default:throw new Error("THREE.MathUtils: Invalid component type.")}}const w0={DEG2RAD:ho},vu=class vu{constructor(e=0,t=0){this.x=e,this.y=t}get width(){return this.x}set width(e){this.x=e}get height(){return this.y}set height(e){this.y=e}set(e,t){return this.x=e,this.y=t,this}setScalar(e){return this.x=e,this.y=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setComponent(e,t){switch(e){case 0:this.x=t;break;case 1:this.y=t;break;default:throw new Error("THREE.Vector2: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;default:throw new Error("THREE.Vector2: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y)}copy(e){return this.x=e.x,this.y=e.y,this}add(e){return this.x+=e.x,this.y+=e.y,this}addScalar(e){return this.x+=e,this.y+=e,this}addVectors(e,t){return this.x=e.x+t.x,this.y=e.y+t.y,this}addScaledVector(e,t){return this.x+=e.x*t,this.y+=e.y*t,this}sub(e){return this.x-=e.x,this.y-=e.y,this}subScalar(e){return this.x-=e,this.y-=e,this}subVectors(e,t){return this.x=e.x-t.x,this.y=e.y-t.y,this}multiply(e){return this.x*=e.x,this.y*=e.y,this}multiplyScalar(e){return this.x*=e,this.y*=e,this}divide(e){return this.x/=e.x,this.y/=e.y,this}divideScalar(e){return this.multiplyScalar(1/e)}applyMatrix3(e){const t=this.x,n=this.y,r=e.elements;return this.x=r[0]*t+r[3]*n+r[6],this.y=r[1]*t+r[4]*n+r[7],this}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this}clamp(e,t){return this.x=ut(this.x,e.x,t.x),this.y=ut(this.y,e.y,t.y),this}clampScalar(e,t){return this.x=ut(this.x,e,t),this.y=ut(this.y,e,t),this}clampLength(e,t){const n=this.length();return this.divideScalar(n||1).multiplyScalar(ut(n,e,t))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this}negate(){return this.x=-this.x,this.y=-this.y,this}dot(e){return this.x*e.x+this.y*e.y}cross(e){return this.x*e.y-this.y*e.x}lengthSq(){return this.x*this.x+this.y*this.y}length(){return Math.sqrt(this.x*this.x+this.y*this.y)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)}normalize(){return this.divideScalar(this.length()||1)}angle(){return Math.atan2(-this.y,-this.x)+Math.PI}angleTo(e){const t=Math.sqrt(this.lengthSq()*e.lengthSq());if(t===0)return Math.PI/2;const n=this.dot(e)/t;return Math.acos(ut(n,-1,1))}distanceTo(e){return Math.sqrt(this.distanceToSquared(e))}distanceToSquared(e){const t=this.x-e.x,n=this.y-e.y;return t*t+n*n}manhattanDistanceTo(e){return Math.abs(this.x-e.x)+Math.abs(this.y-e.y)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,t){return this.x+=(e.x-this.x)*t,this.y+=(e.y-this.y)*t,this}lerpVectors(e,t,n){return this.x=e.x+(t.x-e.x)*n,this.y=e.y+(t.y-e.y)*n,this}equals(e){return e.x===this.x&&e.y===this.y}fromArray(e,t=0){return this.x=e[t],this.y=e[t+1],this}toArray(e=[],t=0){return e[t]=this.x,e[t+1]=this.y,e}fromBufferAttribute(e,t){return this.x=e.getX(t),this.y=e.getY(t),this}rotateAround(e,t){const n=Math.cos(t),r=Math.sin(t),s=this.x-e.x,a=this.y-e.y;return this.x=s*n-a*r+e.x,this.y=s*r+a*n+e.y,this}random(){return this.x=Math.random(),this.y=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y}};vu.prototype.isVector2=!0;let $e=vu;class vr{constructor(e=0,t=0,n=0,r=1){this.isQuaternion=!0,this._x=e,this._y=t,this._z=n,this._w=r}static slerpFlat(e,t,n,r,s,a,o){let l=n[r+0],c=n[r+1],f=n[r+2],h=n[r+3],u=s[a+0],d=s[a+1],x=s[a+2],E=s[a+3];if(h!==E||l!==u||c!==d||f!==x){let m=l*u+c*d+f*x+h*E;m<0&&(u=-u,d=-d,x=-x,E=-E,m=-m);let p=1-o;if(m<.9995){const T=Math.acos(m),P=Math.sin(T);p=Math.sin(p*T)/P,o=Math.sin(o*T)/P,l=l*p+u*o,c=c*p+d*o,f=f*p+x*o,h=h*p+E*o}else{l=l*p+u*o,c=c*p+d*o,f=f*p+x*o,h=h*p+E*o;const T=1/Math.sqrt(l*l+c*c+f*f+h*h);l*=T,c*=T,f*=T,h*=T}}e[t]=l,e[t+1]=c,e[t+2]=f,e[t+3]=h}static multiplyQuaternionsFlat(e,t,n,r,s,a){const o=n[r],l=n[r+1],c=n[r+2],f=n[r+3],h=s[a],u=s[a+1],d=s[a+2],x=s[a+3];return e[t]=o*x+f*h+l*d-c*u,e[t+1]=l*x+f*u+c*h-o*d,e[t+2]=c*x+f*d+o*u-l*h,e[t+3]=f*x-o*h-l*u-c*d,e}get x(){return this._x}set x(e){this._x=e,this._onChangeCallback()}get y(){return this._y}set y(e){this._y=e,this._onChangeCallback()}get z(){return this._z}set z(e){this._z=e,this._onChangeCallback()}get w(){return this._w}set w(e){this._w=e,this._onChangeCallback()}set(e,t,n,r){return this._x=e,this._y=t,this._z=n,this._w=r,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._w)}copy(e){return this._x=e.x,this._y=e.y,this._z=e.z,this._w=e.w,this._onChangeCallback(),this}setFromEuler(e,t=!0){const n=e._x,r=e._y,s=e._z,a=e._order,o=Math.cos,l=Math.sin,c=o(n/2),f=o(r/2),h=o(s/2),u=l(n/2),d=l(r/2),x=l(s/2);switch(a){case"XYZ":this._x=u*f*h+c*d*x,this._y=c*d*h-u*f*x,this._z=c*f*x+u*d*h,this._w=c*f*h-u*d*x;break;case"YXZ":this._x=u*f*h+c*d*x,this._y=c*d*h-u*f*x,this._z=c*f*x-u*d*h,this._w=c*f*h+u*d*x;break;case"ZXY":this._x=u*f*h-c*d*x,this._y=c*d*h+u*f*x,this._z=c*f*x+u*d*h,this._w=c*f*h-u*d*x;break;case"ZYX":this._x=u*f*h-c*d*x,this._y=c*d*h+u*f*x,this._z=c*f*x-u*d*h,this._w=c*f*h+u*d*x;break;case"YZX":this._x=u*f*h+c*d*x,this._y=c*d*h+u*f*x,this._z=c*f*x-u*d*h,this._w=c*f*h-u*d*x;break;case"XZY":this._x=u*f*h-c*d*x,this._y=c*d*h-u*f*x,this._z=c*f*x+u*d*h,this._w=c*f*h+u*d*x;break;default:Je("Quaternion: .setFromEuler() encountered an unknown order: "+a)}return t===!0&&this._onChangeCallback(),this}setFromAxisAngle(e,t){const n=t/2,r=Math.sin(n);return this._x=e.x*r,this._y=e.y*r,this._z=e.z*r,this._w=Math.cos(n),this._onChangeCallback(),this}setFromRotationMatrix(e){const t=e.elements,n=t[0],r=t[4],s=t[8],a=t[1],o=t[5],l=t[9],c=t[2],f=t[6],h=t[10],u=n+o+h;if(u>0){const d=.5/Math.sqrt(u+1);this._w=.25/d,this._x=(f-l)*d,this._y=(s-c)*d,this._z=(a-r)*d}else if(n>o&&n>h){const d=2*Math.sqrt(1+n-o-h);this._w=(f-l)/d,this._x=.25*d,this._y=(r+a)/d,this._z=(s+c)/d}else if(o>h){const d=2*Math.sqrt(1+o-n-h);this._w=(s-c)/d,this._x=(r+a)/d,this._y=.25*d,this._z=(l+f)/d}else{const d=2*Math.sqrt(1+h-n-o);this._w=(a-r)/d,this._x=(s+c)/d,this._y=(l+f)/d,this._z=.25*d}return this._onChangeCallback(),this}setFromUnitVectors(e,t){let n=e.dot(t)+1;return n<1e-8?(n=0,Math.abs(e.x)>Math.abs(e.z)?(this._x=-e.y,this._y=e.x,this._z=0,this._w=n):(this._x=0,this._y=-e.z,this._z=e.y,this._w=n)):(this._x=e.y*t.z-e.z*t.y,this._y=e.z*t.x-e.x*t.z,this._z=e.x*t.y-e.y*t.x,this._w=n),this.normalize()}angleTo(e){return 2*Math.acos(Math.abs(ut(this.dot(e),-1,1)))}rotateTowards(e,t){const n=this.angleTo(e);if(n===0)return this;const r=Math.min(1,t/n);return this.slerp(e,r),this}identity(){return this.set(0,0,0,1)}invert(){return this.conjugate()}conjugate(){return this._x*=-1,this._y*=-1,this._z*=-1,this._onChangeCallback(),this}dot(e){return this._x*e._x+this._y*e._y+this._z*e._z+this._w*e._w}lengthSq(){return this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w}length(){return Math.sqrt(this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w)}normalize(){let e=this.length();return e===0?(this._x=0,this._y=0,this._z=0,this._w=1):(e=1/e,this._x=this._x*e,this._y=this._y*e,this._z=this._z*e,this._w=this._w*e),this._onChangeCallback(),this}multiply(e){return this.multiplyQuaternions(this,e)}premultiply(e){return this.multiplyQuaternions(e,this)}multiplyQuaternions(e,t){const n=e._x,r=e._y,s=e._z,a=e._w,o=t._x,l=t._y,c=t._z,f=t._w;return this._x=n*f+a*o+r*c-s*l,this._y=r*f+a*l+s*o-n*c,this._z=s*f+a*c+n*l-r*o,this._w=a*f-n*o-r*l-s*c,this._onChangeCallback(),this}slerp(e,t){let n=e._x,r=e._y,s=e._z,a=e._w,o=this.dot(e);o<0&&(n=-n,r=-r,s=-s,a=-a,o=-o);let l=1-t;if(o<.9995){const c=Math.acos(o),f=Math.sin(c);l=Math.sin(l*c)/f,t=Math.sin(t*c)/f,this._x=this._x*l+n*t,this._y=this._y*l+r*t,this._z=this._z*l+s*t,this._w=this._w*l+a*t,this._onChangeCallback()}else this._x=this._x*l+n*t,this._y=this._y*l+r*t,this._z=this._z*l+s*t,this._w=this._w*l+a*t,this.normalize();return this}slerpQuaternions(e,t,n){return this.copy(e).slerp(t,n)}random(){const e=2*Math.PI*Math.random(),t=2*Math.PI*Math.random(),n=Math.random(),r=Math.sqrt(1-n),s=Math.sqrt(n);return this.set(r*Math.sin(e),r*Math.cos(e),s*Math.sin(t),s*Math.cos(t))}equals(e){return e._x===this._x&&e._y===this._y&&e._z===this._z&&e._w===this._w}fromArray(e,t=0){return this._x=e[t],this._y=e[t+1],this._z=e[t+2],this._w=e[t+3],this._onChangeCallback(),this}toArray(e=[],t=0){return e[t]=this._x,e[t+1]=this._y,e[t+2]=this._z,e[t+3]=this._w,e}fromBufferAttribute(e,t){return this._x=e.getX(t),this._y=e.getY(t),this._z=e.getZ(t),this._w=e.getW(t),this._onChangeCallback(),this}toJSON(){return this.toArray()}_onChange(e){return this._onChangeCallback=e,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._w}}const Su=class Su{constructor(e=0,t=0,n=0){this.x=e,this.y=t,this.z=n}set(e,t,n){return n===void 0&&(n=this.z),this.x=e,this.y=t,this.z=n,this}setScalar(e){return this.x=e,this.y=e,this.z=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setZ(e){return this.z=e,this}setComponent(e,t){switch(e){case 0:this.x=t;break;case 1:this.y=t;break;case 2:this.z=t;break;default:throw new Error("THREE.Vector3: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;case 2:return this.z;default:throw new Error("THREE.Vector3: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y,this.z)}copy(e){return this.x=e.x,this.y=e.y,this.z=e.z,this}add(e){return this.x+=e.x,this.y+=e.y,this.z+=e.z,this}addScalar(e){return this.x+=e,this.y+=e,this.z+=e,this}addVectors(e,t){return this.x=e.x+t.x,this.y=e.y+t.y,this.z=e.z+t.z,this}addScaledVector(e,t){return this.x+=e.x*t,this.y+=e.y*t,this.z+=e.z*t,this}sub(e){return this.x-=e.x,this.y-=e.y,this.z-=e.z,this}subScalar(e){return this.x-=e,this.y-=e,this.z-=e,this}subVectors(e,t){return this.x=e.x-t.x,this.y=e.y-t.y,this.z=e.z-t.z,this}multiply(e){return this.x*=e.x,this.y*=e.y,this.z*=e.z,this}multiplyScalar(e){return this.x*=e,this.y*=e,this.z*=e,this}multiplyVectors(e,t){return this.x=e.x*t.x,this.y=e.y*t.y,this.z=e.z*t.z,this}applyEuler(e){return this.applyQuaternion(Vf.setFromEuler(e))}applyAxisAngle(e,t){return this.applyQuaternion(Vf.setFromAxisAngle(e,t))}applyMatrix3(e){const t=this.x,n=this.y,r=this.z,s=e.elements;return this.x=s[0]*t+s[3]*n+s[6]*r,this.y=s[1]*t+s[4]*n+s[7]*r,this.z=s[2]*t+s[5]*n+s[8]*r,this}applyNormalMatrix(e){return this.applyMatrix3(e).normalize()}applyMatrix4(e){const t=this.x,n=this.y,r=this.z,s=e.elements,a=1/(s[3]*t+s[7]*n+s[11]*r+s[15]);return this.x=(s[0]*t+s[4]*n+s[8]*r+s[12])*a,this.y=(s[1]*t+s[5]*n+s[9]*r+s[13])*a,this.z=(s[2]*t+s[6]*n+s[10]*r+s[14])*a,this}applyQuaternion(e){const t=this.x,n=this.y,r=this.z,s=e.x,a=e.y,o=e.z,l=e.w,c=2*(a*r-o*n),f=2*(o*t-s*r),h=2*(s*n-a*t);return this.x=t+l*c+a*h-o*f,this.y=n+l*f+o*c-s*h,this.z=r+l*h+s*f-a*c,this}project(e){return this.applyMatrix4(e.matrixWorldInverse).applyMatrix4(e.projectionMatrix)}unproject(e){return this.applyMatrix4(e.projectionMatrixInverse).applyMatrix4(e.matrixWorld)}transformDirection(e){const t=this.x,n=this.y,r=this.z,s=e.elements;return this.x=s[0]*t+s[4]*n+s[8]*r,this.y=s[1]*t+s[5]*n+s[9]*r,this.z=s[2]*t+s[6]*n+s[10]*r,this.normalize()}divide(e){return this.x/=e.x,this.y/=e.y,this.z/=e.z,this}divideScalar(e){return this.multiplyScalar(1/e)}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this.z=Math.min(this.z,e.z),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this.z=Math.max(this.z,e.z),this}clamp(e,t){return this.x=ut(this.x,e.x,t.x),this.y=ut(this.y,e.y,t.y),this.z=ut(this.z,e.z,t.z),this}clampScalar(e,t){return this.x=ut(this.x,e,t),this.y=ut(this.y,e,t),this.z=ut(this.z,e,t),this}clampLength(e,t){const n=this.length();return this.divideScalar(n||1).multiplyScalar(ut(n,e,t))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this}dot(e){return this.x*e.x+this.y*e.y+this.z*e.z}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)}normalize(){return this.divideScalar(this.length()||1)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,t){return this.x+=(e.x-this.x)*t,this.y+=(e.y-this.y)*t,this.z+=(e.z-this.z)*t,this}lerpVectors(e,t,n){return this.x=e.x+(t.x-e.x)*n,this.y=e.y+(t.y-e.y)*n,this.z=e.z+(t.z-e.z)*n,this}cross(e){return this.crossVectors(this,e)}crossVectors(e,t){const n=e.x,r=e.y,s=e.z,a=t.x,o=t.y,l=t.z;return this.x=r*l-s*o,this.y=s*a-n*l,this.z=n*o-r*a,this}projectOnVector(e){const t=e.lengthSq();if(t===0)return this.set(0,0,0);const n=e.dot(this)/t;return this.copy(e).multiplyScalar(n)}projectOnPlane(e){return cl.copy(this).projectOnVector(e),this.sub(cl)}reflect(e){return this.sub(cl.copy(e).multiplyScalar(2*this.dot(e)))}angleTo(e){const t=Math.sqrt(this.lengthSq()*e.lengthSq());if(t===0)return Math.PI/2;const n=this.dot(e)/t;return Math.acos(ut(n,-1,1))}distanceTo(e){return Math.sqrt(this.distanceToSquared(e))}distanceToSquared(e){const t=this.x-e.x,n=this.y-e.y,r=this.z-e.z;return t*t+n*n+r*r}manhattanDistanceTo(e){return Math.abs(this.x-e.x)+Math.abs(this.y-e.y)+Math.abs(this.z-e.z)}setFromSpherical(e){return this.setFromSphericalCoords(e.radius,e.phi,e.theta)}setFromSphericalCoords(e,t,n){const r=Math.sin(t)*e;return this.x=r*Math.sin(n),this.y=Math.cos(t)*e,this.z=r*Math.cos(n),this}setFromCylindrical(e){return this.setFromCylindricalCoords(e.radius,e.theta,e.y)}setFromCylindricalCoords(e,t,n){return this.x=e*Math.sin(t),this.y=n,this.z=e*Math.cos(t),this}setFromMatrixPosition(e){const t=e.elements;return this.x=t[12],this.y=t[13],this.z=t[14],this}setFromMatrixScale(e){const t=this.setFromMatrixColumn(e,0).length(),n=this.setFromMatrixColumn(e,1).length(),r=this.setFromMatrixColumn(e,2).length();return this.x=t,this.y=n,this.z=r,this}setFromMatrixColumn(e,t){return this.fromArray(e.elements,t*4)}setFromMatrix3Column(e,t){return this.fromArray(e.elements,t*3)}setFromEuler(e){return this.x=e._x,this.y=e._y,this.z=e._z,this}setFromColor(e){return this.x=e.r,this.y=e.g,this.z=e.b,this}equals(e){return e.x===this.x&&e.y===this.y&&e.z===this.z}fromArray(e,t=0){return this.x=e[t],this.y=e[t+1],this.z=e[t+2],this}toArray(e=[],t=0){return e[t]=this.x,e[t+1]=this.y,e[t+2]=this.z,e}fromBufferAttribute(e,t){return this.x=e.getX(t),this.y=e.getY(t),this.z=e.getZ(t),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this}randomDirection(){const e=Math.random()*Math.PI*2,t=Math.random()*2-1,n=Math.sqrt(1-t*t);return this.x=n*Math.cos(e),this.y=t,this.z=n*Math.sin(e),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z}};Su.prototype.isVector3=!0;let X=Su;const cl=new X,Vf=new vr,Mu=class Mu{constructor(e,t,n,r,s,a,o,l,c){this.elements=[1,0,0,0,1,0,0,0,1],e!==void 0&&this.set(e,t,n,r,s,a,o,l,c)}set(e,t,n,r,s,a,o,l,c){const f=this.elements;return f[0]=e,f[1]=r,f[2]=o,f[3]=t,f[4]=s,f[5]=l,f[6]=n,f[7]=a,f[8]=c,this}identity(){return this.set(1,0,0,0,1,0,0,0,1),this}copy(e){const t=this.elements,n=e.elements;return t[0]=n[0],t[1]=n[1],t[2]=n[2],t[3]=n[3],t[4]=n[4],t[5]=n[5],t[6]=n[6],t[7]=n[7],t[8]=n[8],this}extractBasis(e,t,n){return e.setFromMatrix3Column(this,0),t.setFromMatrix3Column(this,1),n.setFromMatrix3Column(this,2),this}setFromMatrix4(e){const t=e.elements;return this.set(t[0],t[4],t[8],t[1],t[5],t[9],t[2],t[6],t[10]),this}multiply(e){return this.multiplyMatrices(this,e)}premultiply(e){return this.multiplyMatrices(e,this)}multiplyMatrices(e,t){const n=e.elements,r=t.elements,s=this.elements,a=n[0],o=n[3],l=n[6],c=n[1],f=n[4],h=n[7],u=n[2],d=n[5],x=n[8],E=r[0],m=r[3],p=r[6],T=r[1],P=r[4],S=r[7],C=r[2],y=r[5],I=r[8];return s[0]=a*E+o*T+l*C,s[3]=a*m+o*P+l*y,s[6]=a*p+o*S+l*I,s[1]=c*E+f*T+h*C,s[4]=c*m+f*P+h*y,s[7]=c*p+f*S+h*I,s[2]=u*E+d*T+x*C,s[5]=u*m+d*P+x*y,s[8]=u*p+d*S+x*I,this}multiplyScalar(e){const t=this.elements;return t[0]*=e,t[3]*=e,t[6]*=e,t[1]*=e,t[4]*=e,t[7]*=e,t[2]*=e,t[5]*=e,t[8]*=e,this}determinant(){const e=this.elements,t=e[0],n=e[1],r=e[2],s=e[3],a=e[4],o=e[5],l=e[6],c=e[7],f=e[8];return t*a*f-t*o*c-n*s*f+n*o*l+r*s*c-r*a*l}invert(){const e=this.elements,t=e[0],n=e[1],r=e[2],s=e[3],a=e[4],o=e[5],l=e[6],c=e[7],f=e[8],h=f*a-o*c,u=o*l-f*s,d=c*s-a*l,x=t*h+n*u+r*d;if(x===0)return this.set(0,0,0,0,0,0,0,0,0);const E=1/x;return e[0]=h*E,e[1]=(r*c-f*n)*E,e[2]=(o*n-r*a)*E,e[3]=u*E,e[4]=(f*t-r*l)*E,e[5]=(r*s-o*t)*E,e[6]=d*E,e[7]=(n*l-c*t)*E,e[8]=(a*t-n*s)*E,this}transpose(){let e;const t=this.elements;return e=t[1],t[1]=t[3],t[3]=e,e=t[2],t[2]=t[6],t[6]=e,e=t[5],t[5]=t[7],t[7]=e,this}getNormalMatrix(e){return this.setFromMatrix4(e).invert().transpose()}transposeIntoArray(e){const t=this.elements;return e[0]=t[0],e[1]=t[3],e[2]=t[6],e[3]=t[1],e[4]=t[4],e[5]=t[7],e[6]=t[2],e[7]=t[5],e[8]=t[8],this}setUvTransform(e,t,n,r,s,a,o){const l=Math.cos(s),c=Math.sin(s);return this.set(n*l,n*c,-n*(l*a+c*o)+a+e,-r*c,r*l,-r*(-c*a+l*o)+o+t,0,0,1),this}scale(e,t){return Ts("Matrix3: .scale() is deprecated. Use .makeScale() instead."),this.premultiply(ul.makeScale(e,t)),this}rotate(e){return Ts("Matrix3: .rotate() is deprecated. Use .makeRotation() instead."),this.premultiply(ul.makeRotation(-e)),this}translate(e,t){return Ts("Matrix3: .translate() is deprecated. Use .makeTranslation() instead."),this.premultiply(ul.makeTranslation(e,t)),this}makeTranslation(e,t){return e.isVector2?this.set(1,0,e.x,0,1,e.y,0,0,1):this.set(1,0,e,0,1,t,0,0,1),this}makeRotation(e){const t=Math.cos(e),n=Math.sin(e);return this.set(t,-n,0,n,t,0,0,0,1),this}makeScale(e,t){return this.set(e,0,0,0,t,0,0,0,1),this}equals(e){const t=this.elements,n=e.elements;for(let r=0;r<9;r++)if(t[r]!==n[r])return!1;return!0}fromArray(e,t=0){for(let n=0;n<9;n++)this.elements[n]=e[n+t];return this}toArray(e=[],t=0){const n=this.elements;return e[t]=n[0],e[t+1]=n[1],e[t+2]=n[2],e[t+3]=n[3],e[t+4]=n[4],e[t+5]=n[5],e[t+6]=n[6],e[t+7]=n[7],e[t+8]=n[8],e}clone(){return new this.constructor().fromArray(this.elements)}};Mu.prototype.isMatrix3=!0;let nt=Mu;const ul=new nt,Gf=new nt().set(.4123908,.3575843,.1804808,.212639,.7151687,.0721923,.0193308,.1191948,.9505322),Hf=new nt().set(3.2409699,-1.5373832,-.4986108,-.9692436,1.8759675,.0415551,.0556301,-.203977,1.0569715);function R0(){const i={enabled:!0,workingColorSpace:Mo,spaces:{},convert:function(r,s,a){return this.enabled===!1||s===a||!s||!a||(this.spaces[s].transfer===Ct&&(r.r=$i(r.r),r.g=$i(r.g),r.b=$i(r.b)),this.spaces[s].primaries!==this.spaces[a].primaries&&(r.applyMatrix3(this.spaces[s].toXYZ),r.applyMatrix3(this.spaces[a].fromXYZ)),this.spaces[a].transfer===Ct&&(r.r=As(r.r),r.g=As(r.g),r.b=As(r.b))),r},workingToColorSpace:function(r,s){return this.convert(r,this.workingColorSpace,s)},colorSpaceToWorking:function(r,s){return this.convert(r,s,this.workingColorSpace)},getPrimaries:function(r){return this.spaces[r].primaries},getTransfer:function(r){return r===dr?yo:this.spaces[r].transfer},getToneMappingMode:function(r){return this.spaces[r].outputColorSpaceConfig.toneMappingMode||"standard"},getLuminanceCoefficients:function(r,s=this.workingColorSpace){return r.fromArray(this.spaces[s].luminanceCoefficients)},define:function(r){Object.assign(this.spaces,r)},_getMatrix:function(r,s,a){return r.copy(this.spaces[s].toXYZ).multiply(this.spaces[a].fromXYZ)},_getDrawingBufferColorSpace:function(r){return this.spaces[r].outputColorSpaceConfig.drawingBufferColorSpace},_getUnpackColorSpace:function(r=this.workingColorSpace){return this.spaces[r].workingColorSpaceConfig.unpackColorSpace},fromWorkingColorSpace:function(r,s){return Ts("ColorManagement: .fromWorkingColorSpace() has been renamed to .workingToColorSpace()."),i.workingToColorSpace(r,s)},toWorkingColorSpace:function(r,s){return Ts("ColorManagement: .toWorkingColorSpace() has been renamed to .colorSpaceToWorking()."),i.colorSpaceToWorking(r,s)}},e=[.64,.33,.3,.6,.15,.06],t=[.2126,.7152,.0722],n=[.3127,.329];return i.define({[Mo]:{primaries:e,whitePoint:n,transfer:yo,toXYZ:Gf,fromXYZ:Hf,luminanceCoefficients:t,workingColorSpaceConfig:{unpackColorSpace:ai},outputColorSpaceConfig:{drawingBufferColorSpace:ai}},[ai]:{primaries:e,whitePoint:n,transfer:Ct,toXYZ:Gf,fromXYZ:Hf,luminanceCoefficients:t,outputColorSpaceConfig:{drawingBufferColorSpace:ai}}}),i}const mt=R0();function $i(i){return i<.04045?i*.0773993808:Math.pow(i*.9478672986+.0521327014,2.4)}function As(i){return i<.0031308?i*12.92:1.055*Math.pow(i,.41666)-.055}let is;class C0{static getDataURL(e,t="image/png"){if(/^data:/i.test(e.src)||typeof HTMLCanvasElement>"u")return e.src;let n;if(e instanceof HTMLCanvasElement)n=e;else{is===void 0&&(is=Eo("canvas")),is.width=e.width,is.height=e.height;const r=is.getContext("2d");e instanceof ImageData?r.putImageData(e,0,0):r.drawImage(e,0,0,e.width,e.height),n=is}return n.toDataURL(t)}static sRGBToLinear(e){if(typeof HTMLImageElement<"u"&&e instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&e instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&e instanceof ImageBitmap){const t=Eo("canvas");t.width=e.width,t.height=e.height;const n=t.getContext("2d");n.drawImage(e,0,0,e.width,e.height);const r=n.getImageData(0,0,e.width,e.height),s=r.data;for(let a=0;a<s.length;a++)s[a]=$i(s[a]/255)*255;return n.putImageData(r,0,0),t}else if(e.data){const t=e.data.slice(0);for(let n=0;n<t.length;n++)t instanceof Uint8Array||t instanceof Uint8ClampedArray?t[n]=Math.floor($i(t[n]/255)*255):t[n]=$i(t[n]);return{data:t,width:e.width,height:e.height}}else return Je("ImageUtils.sRGBToLinear(): Unsupported image type. No color space conversion applied."),e}}let P0=0;class du{constructor(e=null){this.isSource=!0,Object.defineProperty(this,"id",{value:P0++}),this.uuid=gr(),this.data=e,this.dataReady=!0,this.version=0}getSize(e){const t=this.data;return typeof HTMLVideoElement<"u"&&t instanceof HTMLVideoElement?e.set(t.videoWidth,t.videoHeight,0):typeof VideoFrame<"u"&&t instanceof VideoFrame?e.set(t.displayWidth,t.displayHeight,0):t!==null?e.set(t.width,t.height,t.depth||0):e.set(0,0,0),e}set needsUpdate(e){e===!0&&this.version++}toJSON(e){const t=e===void 0||typeof e=="string";if(!t&&e.images[this.uuid]!==void 0)return e.images[this.uuid];const n={uuid:this.uuid,url:""},r=this.data;if(r!==null){let s;if(Array.isArray(r)){s=[];for(let a=0,o=r.length;a<o;a++)r[a].isDataTexture?s.push(fl(r[a].image)):s.push(fl(r[a]))}else s=fl(r);n.url=s}return t||(e.images[this.uuid]=n),n}}function fl(i){return typeof HTMLImageElement<"u"&&i instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&i instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&i instanceof ImageBitmap?C0.getDataURL(i):i.data?{data:Array.from(i.data),width:i.width,height:i.height,type:i.data.constructor.name}:(Je("Texture: Unable to serialize Texture."),{})}let D0=0;const hl=new X;class wn extends Sr{constructor(e=wn.DEFAULT_IMAGE,t=wn.DEFAULT_MAPPING,n=Xi,r=Xi,s=yn,a=Fr,o=xi,l=Jn,c=wn.DEFAULT_ANISOTROPY,f=dr){super(),this.isTexture=!0,Object.defineProperty(this,"id",{value:D0++}),this.uuid=gr(),this.name="",this.source=new du(e),this.mipmaps=[],this.mapping=t,this.channel=0,this.wrapS=n,this.wrapT=r,this.magFilter=s,this.minFilter=a,this.anisotropy=c,this.format=o,this.internalFormat=null,this.type=l,this.offset=new $e(0,0),this.repeat=new $e(1,1),this.center=new $e(0,0),this.rotation=0,this.matrixAutoUpdate=!0,this.matrix=new nt,this.generateMipmaps=!0,this.premultiplyAlpha=!1,this.flipY=!0,this.unpackAlignment=4,this.colorSpace=f,this.userData={},this.updateRanges=[],this.version=0,this.onUpdate=null,this.renderTarget=null,this.isRenderTargetTexture=!1,this.isArrayTexture=!!(e&&e.depth&&e.depth>1),this.pmremVersion=0,this.normalized=!1}get width(){return this.source.getSize(hl).x}get height(){return this.source.getSize(hl).y}get depth(){return this.source.getSize(hl).z}get image(){return this.source.data}set image(e){this.source.data=e}updateMatrix(){this.matrix.setUvTransform(this.offset.x,this.offset.y,this.repeat.x,this.repeat.y,this.rotation,this.center.x,this.center.y)}addUpdateRange(e,t){this.updateRanges.push({start:e,count:t})}clearUpdateRanges(){this.updateRanges.length=0}clone(){return new this.constructor().copy(this)}copy(e){return this.name=e.name,this.source=e.source,this.mipmaps=e.mipmaps.slice(0),this.mapping=e.mapping,this.channel=e.channel,this.wrapS=e.wrapS,this.wrapT=e.wrapT,this.magFilter=e.magFilter,this.minFilter=e.minFilter,this.anisotropy=e.anisotropy,this.format=e.format,this.internalFormat=e.internalFormat,this.type=e.type,this.normalized=e.normalized,this.offset.copy(e.offset),this.repeat.copy(e.repeat),this.center.copy(e.center),this.rotation=e.rotation,this.matrixAutoUpdate=e.matrixAutoUpdate,this.matrix.copy(e.matrix),this.generateMipmaps=e.generateMipmaps,this.premultiplyAlpha=e.premultiplyAlpha,this.flipY=e.flipY,this.unpackAlignment=e.unpackAlignment,this.colorSpace=e.colorSpace,this.renderTarget=e.renderTarget,this.isRenderTargetTexture=e.isRenderTargetTexture,this.isArrayTexture=e.isArrayTexture,this.userData=JSON.parse(JSON.stringify(e.userData)),this.needsUpdate=!0,this}setValues(e){for(const t in e){const n=e[t];if(n===void 0){Je(`Texture.setValues(): parameter '${t}' has value of undefined.`);continue}const r=this[t];if(r===void 0){Je(`Texture.setValues(): property '${t}' does not exist.`);continue}r&&n&&r.isVector2&&n.isVector2||r&&n&&r.isVector3&&n.isVector3||r&&n&&r.isMatrix3&&n.isMatrix3?r.copy(n):this[t]=n}}toJSON(e){const t=e===void 0||typeof e=="string";if(!t&&e.textures[this.uuid]!==void 0)return e.textures[this.uuid];const n={metadata:{version:4.7,type:"Texture",generator:"Texture.toJSON"},uuid:this.uuid,name:this.name,image:this.source.toJSON(e).uuid,mapping:this.mapping,channel:this.channel,repeat:[this.repeat.x,this.repeat.y],offset:[this.offset.x,this.offset.y],center:[this.center.x,this.center.y],rotation:this.rotation,wrap:[this.wrapS,this.wrapT],format:this.format,internalFormat:this.internalFormat,type:this.type,normalized:this.normalized,colorSpace:this.colorSpace,minFilter:this.minFilter,magFilter:this.magFilter,anisotropy:this.anisotropy,flipY:this.flipY,generateMipmaps:this.generateMipmaps,premultiplyAlpha:this.premultiplyAlpha,unpackAlignment:this.unpackAlignment};return Object.keys(this.userData).length>0&&(n.userData=this.userData),t||(e.textures[this.uuid]=n),n}dispose(){this.dispatchEvent({type:"dispose"})}transformUv(e){if(this.mapping!==Bd)return e;if(e.applyMatrix3(this.matrix),e.x<0||e.x>1)switch(this.wrapS){case lc:e.x=e.x-Math.floor(e.x);break;case Xi:e.x=e.x<0?0:1;break;case cc:Math.abs(Math.floor(e.x)%2)===1?e.x=Math.ceil(e.x)-e.x:e.x=e.x-Math.floor(e.x);break}if(e.y<0||e.y>1)switch(this.wrapT){case lc:e.y=e.y-Math.floor(e.y);break;case Xi:e.y=e.y<0?0:1;break;case cc:Math.abs(Math.floor(e.y)%2)===1?e.y=Math.ceil(e.y)-e.y:e.y=e.y-Math.floor(e.y);break}return this.flipY&&(e.y=1-e.y),e}set needsUpdate(e){e===!0&&(this.version++,this.source.needsUpdate=!0)}set needsPMREMUpdate(e){e===!0&&this.pmremVersion++}}wn.DEFAULT_IMAGE=null;wn.DEFAULT_MAPPING=Bd;wn.DEFAULT_ANISOTROPY=1;const yu=class yu{constructor(e=0,t=0,n=0,r=1){this.x=e,this.y=t,this.z=n,this.w=r}get width(){return this.z}set width(e){this.z=e}get height(){return this.w}set height(e){this.w=e}set(e,t,n,r){return this.x=e,this.y=t,this.z=n,this.w=r,this}setScalar(e){return this.x=e,this.y=e,this.z=e,this.w=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setZ(e){return this.z=e,this}setW(e){return this.w=e,this}setComponent(e,t){switch(e){case 0:this.x=t;break;case 1:this.y=t;break;case 2:this.z=t;break;case 3:this.w=t;break;default:throw new Error("THREE.Vector4: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;case 2:return this.z;case 3:return this.w;default:throw new Error("THREE.Vector4: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y,this.z,this.w)}copy(e){return this.x=e.x,this.y=e.y,this.z=e.z,this.w=e.w!==void 0?e.w:1,this}add(e){return this.x+=e.x,this.y+=e.y,this.z+=e.z,this.w+=e.w,this}addScalar(e){return this.x+=e,this.y+=e,this.z+=e,this.w+=e,this}addVectors(e,t){return this.x=e.x+t.x,this.y=e.y+t.y,this.z=e.z+t.z,this.w=e.w+t.w,this}addScaledVector(e,t){return this.x+=e.x*t,this.y+=e.y*t,this.z+=e.z*t,this.w+=e.w*t,this}sub(e){return this.x-=e.x,this.y-=e.y,this.z-=e.z,this.w-=e.w,this}subScalar(e){return this.x-=e,this.y-=e,this.z-=e,this.w-=e,this}subVectors(e,t){return this.x=e.x-t.x,this.y=e.y-t.y,this.z=e.z-t.z,this.w=e.w-t.w,this}multiply(e){return this.x*=e.x,this.y*=e.y,this.z*=e.z,this.w*=e.w,this}multiplyScalar(e){return this.x*=e,this.y*=e,this.z*=e,this.w*=e,this}applyMatrix4(e){const t=this.x,n=this.y,r=this.z,s=this.w,a=e.elements;return this.x=a[0]*t+a[4]*n+a[8]*r+a[12]*s,this.y=a[1]*t+a[5]*n+a[9]*r+a[13]*s,this.z=a[2]*t+a[6]*n+a[10]*r+a[14]*s,this.w=a[3]*t+a[7]*n+a[11]*r+a[15]*s,this}divide(e){return this.x/=e.x,this.y/=e.y,this.z/=e.z,this.w/=e.w,this}divideScalar(e){return this.multiplyScalar(1/e)}setAxisAngleFromQuaternion(e){this.w=2*Math.acos(e.w);const t=Math.sqrt(1-e.w*e.w);return t<1e-4?(this.x=1,this.y=0,this.z=0):(this.x=e.x/t,this.y=e.y/t,this.z=e.z/t),this}setAxisAngleFromRotationMatrix(e){let t,n,r,s;const l=e.elements,c=l[0],f=l[4],h=l[8],u=l[1],d=l[5],x=l[9],E=l[2],m=l[6],p=l[10];if(Math.abs(f-u)<.01&&Math.abs(h-E)<.01&&Math.abs(x-m)<.01){if(Math.abs(f+u)<.1&&Math.abs(h+E)<.1&&Math.abs(x+m)<.1&&Math.abs(c+d+p-3)<.1)return this.set(1,0,0,0),this;t=Math.PI;const P=(c+1)/2,S=(d+1)/2,C=(p+1)/2,y=(f+u)/4,I=(h+E)/4,v=(x+m)/4;return P>S&&P>C?P<.01?(n=0,r=.707106781,s=.707106781):(n=Math.sqrt(P),r=y/n,s=I/n):S>C?S<.01?(n=.707106781,r=0,s=.707106781):(r=Math.sqrt(S),n=y/r,s=v/r):C<.01?(n=.707106781,r=.707106781,s=0):(s=Math.sqrt(C),n=I/s,r=v/s),this.set(n,r,s,t),this}let T=Math.sqrt((m-x)*(m-x)+(h-E)*(h-E)+(u-f)*(u-f));return Math.abs(T)<.001&&(T=1),this.x=(m-x)/T,this.y=(h-E)/T,this.z=(u-f)/T,this.w=Math.acos((c+d+p-1)/2),this}setFromMatrixPosition(e){const t=e.elements;return this.x=t[12],this.y=t[13],this.z=t[14],this.w=t[15],this}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this.z=Math.min(this.z,e.z),this.w=Math.min(this.w,e.w),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this.z=Math.max(this.z,e.z),this.w=Math.max(this.w,e.w),this}clamp(e,t){return this.x=ut(this.x,e.x,t.x),this.y=ut(this.y,e.y,t.y),this.z=ut(this.z,e.z,t.z),this.w=ut(this.w,e.w,t.w),this}clampScalar(e,t){return this.x=ut(this.x,e,t),this.y=ut(this.y,e,t),this.z=ut(this.z,e,t),this.w=ut(this.w,e,t),this}clampLength(e,t){const n=this.length();return this.divideScalar(n||1).multiplyScalar(ut(n,e,t))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this.w=Math.floor(this.w),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this.w=Math.ceil(this.w),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this.w=Math.round(this.w),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this.w=Math.trunc(this.w),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this.w=-this.w,this}dot(e){return this.x*e.x+this.y*e.y+this.z*e.z+this.w*e.w}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)+Math.abs(this.w)}normalize(){return this.divideScalar(this.length()||1)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,t){return this.x+=(e.x-this.x)*t,this.y+=(e.y-this.y)*t,this.z+=(e.z-this.z)*t,this.w+=(e.w-this.w)*t,this}lerpVectors(e,t,n){return this.x=e.x+(t.x-e.x)*n,this.y=e.y+(t.y-e.y)*n,this.z=e.z+(t.z-e.z)*n,this.w=e.w+(t.w-e.w)*n,this}equals(e){return e.x===this.x&&e.y===this.y&&e.z===this.z&&e.w===this.w}fromArray(e,t=0){return this.x=e[t],this.y=e[t+1],this.z=e[t+2],this.w=e[t+3],this}toArray(e=[],t=0){return e[t]=this.x,e[t+1]=this.y,e[t+2]=this.z,e[t+3]=this.w,e}fromBufferAttribute(e,t){return this.x=e.getX(t),this.y=e.getY(t),this.z=e.getZ(t),this.w=e.getW(t),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this.w=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z,yield this.w}};yu.prototype.isVector4=!0;let $t=yu;class L0 extends Sr{constructor(e=1,t=1,n={}){super(),n=Object.assign({generateMipmaps:!1,internalFormat:null,minFilter:yn,depthBuffer:!0,stencilBuffer:!1,resolveDepthBuffer:!0,resolveStencilBuffer:!0,depthTexture:null,samples:0,count:1,depth:1,multiview:!1,useArrayDepthTexture:!1},n),this.isRenderTarget=!0,this.width=e,this.height=t,this.depth=n.depth,this.scissor=new $t(0,0,e,t),this.scissorTest=!1,this.viewport=new $t(0,0,e,t),this.textures=[];const r={width:e,height:t,depth:n.depth},s=new wn(r),a=n.count;for(let o=0;o<a;o++)this.textures[o]=s.clone(),this.textures[o].isRenderTargetTexture=!0,this.textures[o].renderTarget=this;this._setTextureOptions(n),this.depthBuffer=n.depthBuffer,this.stencilBuffer=n.stencilBuffer,this.resolveDepthBuffer=n.resolveDepthBuffer,this.resolveStencilBuffer=n.resolveStencilBuffer,this._depthTexture=null,this.depthTexture=n.depthTexture,this.samples=n.samples,this.multiview=n.multiview,this.useArrayDepthTexture=n.useArrayDepthTexture}_setTextureOptions(e={}){const t={minFilter:yn,generateMipmaps:!1,flipY:!1,internalFormat:null};e.mapping!==void 0&&(t.mapping=e.mapping),e.wrapS!==void 0&&(t.wrapS=e.wrapS),e.wrapT!==void 0&&(t.wrapT=e.wrapT),e.wrapR!==void 0&&(t.wrapR=e.wrapR),e.magFilter!==void 0&&(t.magFilter=e.magFilter),e.minFilter!==void 0&&(t.minFilter=e.minFilter),e.format!==void 0&&(t.format=e.format),e.type!==void 0&&(t.type=e.type),e.anisotropy!==void 0&&(t.anisotropy=e.anisotropy),e.colorSpace!==void 0&&(t.colorSpace=e.colorSpace),e.flipY!==void 0&&(t.flipY=e.flipY),e.generateMipmaps!==void 0&&(t.generateMipmaps=e.generateMipmaps),e.internalFormat!==void 0&&(t.internalFormat=e.internalFormat);for(let n=0;n<this.textures.length;n++)this.textures[n].setValues(t)}get texture(){return this.textures[0]}set texture(e){this.textures[0]=e}set depthTexture(e){this._depthTexture!==null&&(this._depthTexture.renderTarget=null),e!==null&&(e.renderTarget=this),this._depthTexture=e}get depthTexture(){return this._depthTexture}setSize(e,t,n=1){if(this.width!==e||this.height!==t||this.depth!==n){this.width=e,this.height=t,this.depth=n;for(let r=0,s=this.textures.length;r<s;r++)this.textures[r].image.width=e,this.textures[r].image.height=t,this.textures[r].image.depth=n,this.textures[r].isData3DTexture!==!0&&(this.textures[r].isArrayTexture=this.textures[r].image.depth>1);this.dispose()}this.viewport.set(0,0,e,t),this.scissor.set(0,0,e,t)}clone(){return new this.constructor().copy(this)}copy(e){this.width=e.width,this.height=e.height,this.depth=e.depth,this.scissor.copy(e.scissor),this.scissorTest=e.scissorTest,this.viewport.copy(e.viewport),this.textures.length=0;for(let t=0,n=e.textures.length;t<n;t++){this.textures[t]=e.textures[t].clone(),this.textures[t].isRenderTargetTexture=!0,this.textures[t].renderTarget=this;const r=Object.assign({},e.textures[t].image);this.textures[t].source=new du(r)}return this.depthBuffer=e.depthBuffer,this.stencilBuffer=e.stencilBuffer,this.resolveDepthBuffer=e.resolveDepthBuffer,this.resolveStencilBuffer=e.resolveStencilBuffer,e.depthTexture!==null&&(this.depthTexture=e.depthTexture.clone()),this.samples=e.samples,this.multiview=e.multiview,this.useArrayDepthTexture=e.useArrayDepthTexture,this}dispose(){this.dispatchEvent({type:"dispose"})}}class Ni extends L0{constructor(e=1,t=1,n={}){super(e,t,n),this.isWebGLRenderTarget=!0}}class qd extends wn{constructor(e=null,t=1,n=1,r=1){super(null),this.isDataArrayTexture=!0,this.image={data:e,width:t,height:n,depth:r},this.magFilter=Mn,this.minFilter=Mn,this.wrapR=Xi,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1,this.layerUpdates=new Set}addLayerUpdate(e){this.layerUpdates.add(e)}clearLayerUpdates(){this.layerUpdates.clear()}}class I0 extends wn{constructor(e=null,t=1,n=1,r=1){super(null),this.isData3DTexture=!0,this.image={data:e,width:t,height:n,depth:r},this.magFilter=Mn,this.minFilter=Mn,this.wrapR=Xi,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}}const Ro=class Ro{constructor(e,t,n,r,s,a,o,l,c,f,h,u,d,x,E,m){this.elements=[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1],e!==void 0&&this.set(e,t,n,r,s,a,o,l,c,f,h,u,d,x,E,m)}set(e,t,n,r,s,a,o,l,c,f,h,u,d,x,E,m){const p=this.elements;return p[0]=e,p[4]=t,p[8]=n,p[12]=r,p[1]=s,p[5]=a,p[9]=o,p[13]=l,p[2]=c,p[6]=f,p[10]=h,p[14]=u,p[3]=d,p[7]=x,p[11]=E,p[15]=m,this}identity(){return this.set(1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1),this}clone(){return new Ro().fromArray(this.elements)}copy(e){const t=this.elements,n=e.elements;return t[0]=n[0],t[1]=n[1],t[2]=n[2],t[3]=n[3],t[4]=n[4],t[5]=n[5],t[6]=n[6],t[7]=n[7],t[8]=n[8],t[9]=n[9],t[10]=n[10],t[11]=n[11],t[12]=n[12],t[13]=n[13],t[14]=n[14],t[15]=n[15],this}copyPosition(e){const t=this.elements,n=e.elements;return t[12]=n[12],t[13]=n[13],t[14]=n[14],this}setFromMatrix3(e){const t=e.elements;return this.set(t[0],t[3],t[6],0,t[1],t[4],t[7],0,t[2],t[5],t[8],0,0,0,0,1),this}extractBasis(e,t,n){return this.determinantAffine()===0?(e.set(1,0,0),t.set(0,1,0),n.set(0,0,1),this):(e.setFromMatrixColumn(this,0),t.setFromMatrixColumn(this,1),n.setFromMatrixColumn(this,2),this)}makeBasis(e,t,n){return this.set(e.x,t.x,n.x,0,e.y,t.y,n.y,0,e.z,t.z,n.z,0,0,0,0,1),this}extractRotation(e){if(e.determinantAffine()===0)return this.identity();const t=this.elements,n=e.elements,r=1/rs.setFromMatrixColumn(e,0).length(),s=1/rs.setFromMatrixColumn(e,1).length(),a=1/rs.setFromMatrixColumn(e,2).length();return t[0]=n[0]*r,t[1]=n[1]*r,t[2]=n[2]*r,t[3]=0,t[4]=n[4]*s,t[5]=n[5]*s,t[6]=n[6]*s,t[7]=0,t[8]=n[8]*a,t[9]=n[9]*a,t[10]=n[10]*a,t[11]=0,t[12]=0,t[13]=0,t[14]=0,t[15]=1,this}makeRotationFromEuler(e){const t=this.elements,n=e.x,r=e.y,s=e.z,a=Math.cos(n),o=Math.sin(n),l=Math.cos(r),c=Math.sin(r),f=Math.cos(s),h=Math.sin(s);if(e.order==="XYZ"){const u=a*f,d=a*h,x=o*f,E=o*h;t[0]=l*f,t[4]=-l*h,t[8]=c,t[1]=d+x*c,t[5]=u-E*c,t[9]=-o*l,t[2]=E-u*c,t[6]=x+d*c,t[10]=a*l}else if(e.order==="YXZ"){const u=l*f,d=l*h,x=c*f,E=c*h;t[0]=u+E*o,t[4]=x*o-d,t[8]=a*c,t[1]=a*h,t[5]=a*f,t[9]=-o,t[2]=d*o-x,t[6]=E+u*o,t[10]=a*l}else if(e.order==="ZXY"){const u=l*f,d=l*h,x=c*f,E=c*h;t[0]=u-E*o,t[4]=-a*h,t[8]=x+d*o,t[1]=d+x*o,t[5]=a*f,t[9]=E-u*o,t[2]=-a*c,t[6]=o,t[10]=a*l}else if(e.order==="ZYX"){const u=a*f,d=a*h,x=o*f,E=o*h;t[0]=l*f,t[4]=x*c-d,t[8]=u*c+E,t[1]=l*h,t[5]=E*c+u,t[9]=d*c-x,t[2]=-c,t[6]=o*l,t[10]=a*l}else if(e.order==="YZX"){const u=a*l,d=a*c,x=o*l,E=o*c;t[0]=l*f,t[4]=E-u*h,t[8]=x*h+d,t[1]=h,t[5]=a*f,t[9]=-o*f,t[2]=-c*f,t[6]=d*h+x,t[10]=u-E*h}else if(e.order==="XZY"){const u=a*l,d=a*c,x=o*l,E=o*c;t[0]=l*f,t[4]=-h,t[8]=c*f,t[1]=u*h+E,t[5]=a*f,t[9]=d*h-x,t[2]=x*h-d,t[6]=o*f,t[10]=E*h+u}return t[3]=0,t[7]=0,t[11]=0,t[12]=0,t[13]=0,t[14]=0,t[15]=1,this}makeRotationFromQuaternion(e){return this.compose(U0,e,N0)}lookAt(e,t,n){const r=this.elements;return Kn.subVectors(e,t),Kn.lengthSq()===0&&(Kn.z=1),Kn.normalize(),sr.crossVectors(n,Kn),sr.lengthSq()===0&&(Math.abs(n.z)===1?Kn.x+=1e-4:Kn.z+=1e-4,Kn.normalize(),sr.crossVectors(n,Kn)),sr.normalize(),La.crossVectors(Kn,sr),r[0]=sr.x,r[4]=La.x,r[8]=Kn.x,r[1]=sr.y,r[5]=La.y,r[9]=Kn.y,r[2]=sr.z,r[6]=La.z,r[10]=Kn.z,this}multiply(e){return this.multiplyMatrices(this,e)}premultiply(e){return this.multiplyMatrices(e,this)}multiplyMatrices(e,t){const n=e.elements,r=t.elements,s=this.elements,a=n[0],o=n[4],l=n[8],c=n[12],f=n[1],h=n[5],u=n[9],d=n[13],x=n[2],E=n[6],m=n[10],p=n[14],T=n[3],P=n[7],S=n[11],C=n[15],y=r[0],I=r[4],v=r[8],A=r[12],F=r[1],D=r[5],z=r[9],O=r[13],k=r[2],L=r[6],N=r[10],B=r[14],j=r[3],ne=r[7],ae=r[11],fe=r[15];return s[0]=a*y+o*F+l*k+c*j,s[4]=a*I+o*D+l*L+c*ne,s[8]=a*v+o*z+l*N+c*ae,s[12]=a*A+o*O+l*B+c*fe,s[1]=f*y+h*F+u*k+d*j,s[5]=f*I+h*D+u*L+d*ne,s[9]=f*v+h*z+u*N+d*ae,s[13]=f*A+h*O+u*B+d*fe,s[2]=x*y+E*F+m*k+p*j,s[6]=x*I+E*D+m*L+p*ne,s[10]=x*v+E*z+m*N+p*ae,s[14]=x*A+E*O+m*B+p*fe,s[3]=T*y+P*F+S*k+C*j,s[7]=T*I+P*D+S*L+C*ne,s[11]=T*v+P*z+S*N+C*ae,s[15]=T*A+P*O+S*B+C*fe,this}multiplyScalar(e){const t=this.elements;return t[0]*=e,t[4]*=e,t[8]*=e,t[12]*=e,t[1]*=e,t[5]*=e,t[9]*=e,t[13]*=e,t[2]*=e,t[6]*=e,t[10]*=e,t[14]*=e,t[3]*=e,t[7]*=e,t[11]*=e,t[15]*=e,this}determinant(){const e=this.elements,t=e[0],n=e[4],r=e[8],s=e[12],a=e[1],o=e[5],l=e[9],c=e[13],f=e[2],h=e[6],u=e[10],d=e[14],x=e[3],E=e[7],m=e[11],p=e[15],T=l*d-c*u,P=o*d-c*h,S=o*u-l*h,C=a*d-c*f,y=a*u-l*f,I=a*h-o*f;return t*(E*T-m*P+p*S)-n*(x*T-m*C+p*y)+r*(x*P-E*C+p*I)-s*(x*S-E*y+m*I)}determinantAffine(){const e=this.elements,t=e[0],n=e[4],r=e[8],s=e[1],a=e[5],o=e[9],l=e[2],c=e[6],f=e[10];return t*(a*f-o*c)-n*(s*f-o*l)+r*(s*c-a*l)}transpose(){const e=this.elements;let t;return t=e[1],e[1]=e[4],e[4]=t,t=e[2],e[2]=e[8],e[8]=t,t=e[6],e[6]=e[9],e[9]=t,t=e[3],e[3]=e[12],e[12]=t,t=e[7],e[7]=e[13],e[13]=t,t=e[11],e[11]=e[14],e[14]=t,this}setPosition(e,t,n){const r=this.elements;return e.isVector3?(r[12]=e.x,r[13]=e.y,r[14]=e.z):(r[12]=e,r[13]=t,r[14]=n),this}invert(){const e=this.elements,t=e[0],n=e[1],r=e[2],s=e[3],a=e[4],o=e[5],l=e[6],c=e[7],f=e[8],h=e[9],u=e[10],d=e[11],x=e[12],E=e[13],m=e[14],p=e[15],T=t*o-n*a,P=t*l-r*a,S=t*c-s*a,C=n*l-r*o,y=n*c-s*o,I=r*c-s*l,v=f*E-h*x,A=f*m-u*x,F=f*p-d*x,D=h*m-u*E,z=h*p-d*E,O=u*p-d*m,k=T*O-P*z+S*D+C*F-y*A+I*v;if(k===0)return this.set(0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0);const L=1/k;return e[0]=(o*O-l*z+c*D)*L,e[1]=(r*z-n*O-s*D)*L,e[2]=(E*I-m*y+p*C)*L,e[3]=(u*y-h*I-d*C)*L,e[4]=(l*F-a*O-c*A)*L,e[5]=(t*O-r*F+s*A)*L,e[6]=(m*S-x*I-p*P)*L,e[7]=(f*I-u*S+d*P)*L,e[8]=(a*z-o*F+c*v)*L,e[9]=(n*F-t*z-s*v)*L,e[10]=(x*y-E*S+p*T)*L,e[11]=(h*S-f*y-d*T)*L,e[12]=(o*A-a*D-l*v)*L,e[13]=(t*D-n*A+r*v)*L,e[14]=(E*P-x*C-m*T)*L,e[15]=(f*C-h*P+u*T)*L,this}scale(e){const t=this.elements,n=e.x,r=e.y,s=e.z;return t[0]*=n,t[4]*=r,t[8]*=s,t[1]*=n,t[5]*=r,t[9]*=s,t[2]*=n,t[6]*=r,t[10]*=s,t[3]*=n,t[7]*=r,t[11]*=s,this}getMaxScaleOnAxis(){const e=this.elements,t=e[0]*e[0]+e[1]*e[1]+e[2]*e[2],n=e[4]*e[4]+e[5]*e[5]+e[6]*e[6],r=e[8]*e[8]+e[9]*e[9]+e[10]*e[10];return Math.sqrt(Math.max(t,n,r))}makeTranslation(e,t,n){return e.isVector3?this.set(1,0,0,e.x,0,1,0,e.y,0,0,1,e.z,0,0,0,1):this.set(1,0,0,e,0,1,0,t,0,0,1,n,0,0,0,1),this}makeRotationX(e){const t=Math.cos(e),n=Math.sin(e);return this.set(1,0,0,0,0,t,-n,0,0,n,t,0,0,0,0,1),this}makeRotationY(e){const t=Math.cos(e),n=Math.sin(e);return this.set(t,0,n,0,0,1,0,0,-n,0,t,0,0,0,0,1),this}makeRotationZ(e){const t=Math.cos(e),n=Math.sin(e);return this.set(t,-n,0,0,n,t,0,0,0,0,1,0,0,0,0,1),this}makeRotationAxis(e,t){const n=Math.cos(t),r=Math.sin(t),s=1-n,a=e.x,o=e.y,l=e.z,c=s*a,f=s*o;return this.set(c*a+n,c*o-r*l,c*l+r*o,0,c*o+r*l,f*o+n,f*l-r*a,0,c*l-r*o,f*l+r*a,s*l*l+n,0,0,0,0,1),this}makeScale(e,t,n){return this.set(e,0,0,0,0,t,0,0,0,0,n,0,0,0,0,1),this}makeShear(e,t,n,r,s,a){return this.set(1,n,s,0,e,1,a,0,t,r,1,0,0,0,0,1),this}compose(e,t,n){const r=this.elements,s=t._x,a=t._y,o=t._z,l=t._w,c=s+s,f=a+a,h=o+o,u=s*c,d=s*f,x=s*h,E=a*f,m=a*h,p=o*h,T=l*c,P=l*f,S=l*h,C=n.x,y=n.y,I=n.z;return r[0]=(1-(E+p))*C,r[1]=(d+S)*C,r[2]=(x-P)*C,r[3]=0,r[4]=(d-S)*y,r[5]=(1-(u+p))*y,r[6]=(m+T)*y,r[7]=0,r[8]=(x+P)*I,r[9]=(m-T)*I,r[10]=(1-(u+E))*I,r[11]=0,r[12]=e.x,r[13]=e.y,r[14]=e.z,r[15]=1,this}decompose(e,t,n){const r=this.elements;e.x=r[12],e.y=r[13],e.z=r[14];const s=this.determinantAffine();if(s===0)return n.set(1,1,1),t.identity(),this;let a=rs.set(r[0],r[1],r[2]).length();const o=rs.set(r[4],r[5],r[6]).length(),l=rs.set(r[8],r[9],r[10]).length();s<0&&(a=-a),hi.copy(this);const c=1/a,f=1/o,h=1/l;return hi.elements[0]*=c,hi.elements[1]*=c,hi.elements[2]*=c,hi.elements[4]*=f,hi.elements[5]*=f,hi.elements[6]*=f,hi.elements[8]*=h,hi.elements[9]*=h,hi.elements[10]*=h,t.setFromRotationMatrix(hi),n.x=a,n.y=o,n.z=l,this}makePerspective(e,t,n,r,s,a,o=Ii,l=!1){const c=this.elements,f=2*s/(t-e),h=2*s/(n-r),u=(t+e)/(t-e),d=(n+r)/(n-r);let x,E;if(l)x=s/(a-s),E=a*s/(a-s);else if(o===Ii)x=-(a+s)/(a-s),E=-2*a*s/(a-s);else if(o===va)x=-a/(a-s),E=-a*s/(a-s);else throw new Error("THREE.Matrix4.makePerspective(): Invalid coordinate system: "+o);return c[0]=f,c[4]=0,c[8]=u,c[12]=0,c[1]=0,c[5]=h,c[9]=d,c[13]=0,c[2]=0,c[6]=0,c[10]=x,c[14]=E,c[3]=0,c[7]=0,c[11]=-1,c[15]=0,this}makeOrthographic(e,t,n,r,s,a,o=Ii,l=!1){const c=this.elements,f=2/(t-e),h=2/(n-r),u=-(t+e)/(t-e),d=-(n+r)/(n-r);let x,E;if(l)x=1/(a-s),E=a/(a-s);else if(o===Ii)x=-2/(a-s),E=-(a+s)/(a-s);else if(o===va)x=-1/(a-s),E=-s/(a-s);else throw new Error("THREE.Matrix4.makeOrthographic(): Invalid coordinate system: "+o);return c[0]=f,c[4]=0,c[8]=0,c[12]=u,c[1]=0,c[5]=h,c[9]=0,c[13]=d,c[2]=0,c[6]=0,c[10]=x,c[14]=E,c[3]=0,c[7]=0,c[11]=0,c[15]=1,this}equals(e){const t=this.elements,n=e.elements;for(let r=0;r<16;r++)if(t[r]!==n[r])return!1;return!0}fromArray(e,t=0){for(let n=0;n<16;n++)this.elements[n]=e[n+t];return this}toArray(e=[],t=0){const n=this.elements;return e[t]=n[0],e[t+1]=n[1],e[t+2]=n[2],e[t+3]=n[3],e[t+4]=n[4],e[t+5]=n[5],e[t+6]=n[6],e[t+7]=n[7],e[t+8]=n[8],e[t+9]=n[9],e[t+10]=n[10],e[t+11]=n[11],e[t+12]=n[12],e[t+13]=n[13],e[t+14]=n[14],e[t+15]=n[15],e}};Ro.prototype.isMatrix4=!0;let Xt=Ro;const rs=new X,hi=new Xt,U0=new X(0,0,0),N0=new X(1,1,1),sr=new X,La=new X,Kn=new X,Wf=new Xt,Xf=new vr;class Gr{constructor(e=0,t=0,n=0,r=Gr.DEFAULT_ORDER){this.isEuler=!0,this._x=e,this._y=t,this._z=n,this._order=r}get x(){return this._x}set x(e){this._x=e,this._onChangeCallback()}get y(){return this._y}set y(e){this._y=e,this._onChangeCallback()}get z(){return this._z}set z(e){this._z=e,this._onChangeCallback()}get order(){return this._order}set order(e){this._order=e,this._onChangeCallback()}set(e,t,n,r=this._order){return this._x=e,this._y=t,this._z=n,this._order=r,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._order)}copy(e){return this._x=e._x,this._y=e._y,this._z=e._z,this._order=e._order,this._onChangeCallback(),this}setFromRotationMatrix(e,t=this._order,n=!0){const r=e.elements,s=r[0],a=r[4],o=r[8],l=r[1],c=r[5],f=r[9],h=r[2],u=r[6],d=r[10];switch(t){case"XYZ":this._y=Math.asin(ut(o,-1,1)),Math.abs(o)<.9999999?(this._x=Math.atan2(-f,d),this._z=Math.atan2(-a,s)):(this._x=Math.atan2(u,c),this._z=0);break;case"YXZ":this._x=Math.asin(-ut(f,-1,1)),Math.abs(f)<.9999999?(this._y=Math.atan2(o,d),this._z=Math.atan2(l,c)):(this._y=Math.atan2(-h,s),this._z=0);break;case"ZXY":this._x=Math.asin(ut(u,-1,1)),Math.abs(u)<.9999999?(this._y=Math.atan2(-h,d),this._z=Math.atan2(-a,c)):(this._y=0,this._z=Math.atan2(l,s));break;case"ZYX":this._y=Math.asin(-ut(h,-1,1)),Math.abs(h)<.9999999?(this._x=Math.atan2(u,d),this._z=Math.atan2(l,s)):(this._x=0,this._z=Math.atan2(-a,c));break;case"YZX":this._z=Math.asin(ut(l,-1,1)),Math.abs(l)<.9999999?(this._x=Math.atan2(-f,c),this._y=Math.atan2(-h,s)):(this._x=0,this._y=Math.atan2(o,d));break;case"XZY":this._z=Math.asin(-ut(a,-1,1)),Math.abs(a)<.9999999?(this._x=Math.atan2(u,c),this._y=Math.atan2(o,s)):(this._x=Math.atan2(-f,d),this._y=0);break;default:Je("Euler: .setFromRotationMatrix() encountered an unknown order: "+t)}return this._order=t,n===!0&&this._onChangeCallback(),this}setFromQuaternion(e,t,n){return Wf.makeRotationFromQuaternion(e),this.setFromRotationMatrix(Wf,t,n)}setFromVector3(e,t=this._order){return this.set(e.x,e.y,e.z,t)}reorder(e){return Xf.setFromEuler(this),this.setFromQuaternion(Xf,e)}equals(e){return e._x===this._x&&e._y===this._y&&e._z===this._z&&e._order===this._order}fromArray(e){return this._x=e[0],this._y=e[1],this._z=e[2],e[3]!==void 0&&(this._order=e[3]),this._onChangeCallback(),this}toArray(e=[],t=0){return e[t]=this._x,e[t+1]=this._y,e[t+2]=this._z,e[t+3]=this._order,e}_onChange(e){return this._onChangeCallback=e,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._order}}Gr.DEFAULT_ORDER="XYZ";class Kd{constructor(){this.mask=1}set(e){this.mask=(1<<e|0)>>>0}enable(e){this.mask|=1<<e|0}enableAll(){this.mask=-1}toggle(e){this.mask^=1<<e|0}disable(e){this.mask&=~(1<<e|0)}disableAll(){this.mask=0}test(e){return(this.mask&e.mask)!==0}isEnabled(e){return(this.mask&(1<<e|0))!==0}}let F0=0;const Yf=new X,ss=new vr,ki=new Xt,Ia=new X,js=new X,O0=new X,B0=new vr,qf=new X(1,0,0),Kf=new X(0,1,0),$f=new X(0,0,1),Zf={type:"added"},z0={type:"removed"},as={type:"childadded",child:null},dl={type:"childremoved",child:null};class _n extends Sr{constructor(){super(),this.isObject3D=!0,Object.defineProperty(this,"id",{value:F0++}),this.uuid=gr(),this.name="",this.type="Object3D",this.parent=null,this.children=[],this.up=_n.DEFAULT_UP.clone();const e=new X,t=new Gr,n=new vr,r=new X(1,1,1);function s(){n.setFromEuler(t,!1)}function a(){t.setFromQuaternion(n,void 0,!1)}t._onChange(s),n._onChange(a),Object.defineProperties(this,{position:{configurable:!0,enumerable:!0,value:e},rotation:{configurable:!0,enumerable:!0,value:t},quaternion:{configurable:!0,enumerable:!0,value:n},scale:{configurable:!0,enumerable:!0,value:r},modelViewMatrix:{value:new Xt},normalMatrix:{value:new nt}}),this.matrix=new Xt,this.matrixWorld=new Xt,this.matrixAutoUpdate=_n.DEFAULT_MATRIX_AUTO_UPDATE,this.matrixWorldAutoUpdate=_n.DEFAULT_MATRIX_WORLD_AUTO_UPDATE,this.matrixWorldNeedsUpdate=!1,this.layers=new Kd,this.visible=!0,this.castShadow=!1,this.receiveShadow=!1,this.frustumCulled=!0,this.renderOrder=0,this.animations=[],this.customDepthMaterial=void 0,this.customDistanceMaterial=void 0,this.static=!1,this.userData={},this.pivot=null}onBeforeShadow(){}onAfterShadow(){}onBeforeRender(){}onAfterRender(){}applyMatrix4(e){this.matrixAutoUpdate&&this.updateMatrix(),this.matrix.premultiply(e),this.matrix.decompose(this.position,this.quaternion,this.scale)}applyQuaternion(e){return this.quaternion.premultiply(e),this}setRotationFromAxisAngle(e,t){this.quaternion.setFromAxisAngle(e,t)}setRotationFromEuler(e){this.quaternion.setFromEuler(e,!0)}setRotationFromMatrix(e){this.quaternion.setFromRotationMatrix(e)}setRotationFromQuaternion(e){this.quaternion.copy(e)}rotateOnAxis(e,t){return ss.setFromAxisAngle(e,t),this.quaternion.multiply(ss),this}rotateOnWorldAxis(e,t){return ss.setFromAxisAngle(e,t),this.quaternion.premultiply(ss),this}rotateX(e){return this.rotateOnAxis(qf,e)}rotateY(e){return this.rotateOnAxis(Kf,e)}rotateZ(e){return this.rotateOnAxis($f,e)}translateOnAxis(e,t){return Yf.copy(e).applyQuaternion(this.quaternion),this.position.add(Yf.multiplyScalar(t)),this}translateX(e){return this.translateOnAxis(qf,e)}translateY(e){return this.translateOnAxis(Kf,e)}translateZ(e){return this.translateOnAxis($f,e)}localToWorld(e){return this.updateWorldMatrix(!0,!1),e.applyMatrix4(this.matrixWorld)}worldToLocal(e){return this.updateWorldMatrix(!0,!1),e.applyMatrix4(ki.copy(this.matrixWorld).invert())}lookAt(e,t,n){e.isVector3?Ia.copy(e):Ia.set(e,t,n);const r=this.parent;this.updateWorldMatrix(!0,!1),js.setFromMatrixPosition(this.matrixWorld),this.isCamera||this.isLight?ki.lookAt(js,Ia,this.up):ki.lookAt(Ia,js,this.up),this.quaternion.setFromRotationMatrix(ki),r&&(ki.extractRotation(r.matrixWorld),ss.setFromRotationMatrix(ki),this.quaternion.premultiply(ss.invert()))}add(e){if(arguments.length>1){for(let t=0;t<arguments.length;t++)this.add(arguments[t]);return this}return e===this?(gt("Object3D.add: object can't be added as a child of itself.",e),this):(e&&e.isObject3D?(e.removeFromParent(),e.parent=this,this.children.push(e),e.dispatchEvent(Zf),as.child=e,this.dispatchEvent(as),as.child=null):gt("Object3D.add: object not an instance of THREE.Object3D.",e),this)}remove(e){if(arguments.length>1){for(let n=0;n<arguments.length;n++)this.remove(arguments[n]);return this}const t=this.children.indexOf(e);return t!==-1&&(e.parent=null,this.children.splice(t,1),e.dispatchEvent(z0),dl.child=e,this.dispatchEvent(dl),dl.child=null),this}removeFromParent(){const e=this.parent;return e!==null&&e.remove(this),this}clear(){return this.remove(...this.children)}attach(e){return this.updateWorldMatrix(!0,!1),ki.copy(this.matrixWorld).invert(),e.parent!==null&&(e.parent.updateWorldMatrix(!0,!1),ki.multiply(e.parent.matrixWorld)),e.applyMatrix4(ki),e.removeFromParent(),e.parent=this,this.children.push(e),e.updateWorldMatrix(!1,!0),e.dispatchEvent(Zf),as.child=e,this.dispatchEvent(as),as.child=null,this}getObjectById(e){return this.getObjectByProperty("id",e)}getObjectByName(e){return this.getObjectByProperty("name",e)}getObjectByProperty(e,t){if(this[e]===t)return this;for(let n=0,r=this.children.length;n<r;n++){const a=this.children[n].getObjectByProperty(e,t);if(a!==void 0)return a}}getObjectsByProperty(e,t,n=[]){this[e]===t&&n.push(this);const r=this.children;for(let s=0,a=r.length;s<a;s++)r[s].getObjectsByProperty(e,t,n);return n}getWorldPosition(e){return this.updateWorldMatrix(!0,!1),e.setFromMatrixPosition(this.matrixWorld)}getWorldQuaternion(e){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(js,e,O0),e}getWorldScale(e){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(js,B0,e),e}getWorldDirection(e){this.updateWorldMatrix(!0,!1);const t=this.matrixWorld.elements;return e.set(t[8],t[9],t[10]).normalize()}raycast(){}traverse(e){e(this);const t=this.children;for(let n=0,r=t.length;n<r;n++)t[n].traverse(e)}traverseVisible(e){if(this.visible===!1)return;e(this);const t=this.children;for(let n=0,r=t.length;n<r;n++)t[n].traverseVisible(e)}traverseAncestors(e){const t=this.parent;t!==null&&(e(t),t.traverseAncestors(e))}updateMatrix(){this.matrix.compose(this.position,this.quaternion,this.scale);const e=this.pivot;if(e!==null){const t=e.x,n=e.y,r=e.z,s=this.matrix.elements;s[12]+=t-s[0]*t-s[4]*n-s[8]*r,s[13]+=n-s[1]*t-s[5]*n-s[9]*r,s[14]+=r-s[2]*t-s[6]*n-s[10]*r}this.matrixWorldNeedsUpdate=!0}updateMatrixWorld(e){this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||e)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,e=!0);const t=this.children;for(let n=0,r=t.length;n<r;n++)t[n].updateMatrixWorld(e)}updateWorldMatrix(e,t,n=!1){const r=this.parent;if(e===!0&&r!==null&&r.updateWorldMatrix(!0,!1),this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||n)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,n=!0),t===!0){const s=this.children;for(let a=0,o=s.length;a<o;a++)s[a].updateWorldMatrix(!1,!0,n)}}toJSON(e){const t=e===void 0||typeof e=="string",n={};t&&(e={geometries:{},materials:{},textures:{},images:{},shapes:{},skeletons:{},animations:{},nodes:{}},n.metadata={version:4.7,type:"Object",generator:"Object3D.toJSON"});const r={};r.uuid=this.uuid,r.type=this.type,this.name!==""&&(r.name=this.name),this.castShadow===!0&&(r.castShadow=!0),this.receiveShadow===!0&&(r.receiveShadow=!0),this.visible===!1&&(r.visible=!1),this.frustumCulled===!1&&(r.frustumCulled=!1),this.renderOrder!==0&&(r.renderOrder=this.renderOrder),this.static!==!1&&(r.static=this.static),Object.keys(this.userData).length>0&&(r.userData=this.userData),r.layers=this.layers.mask,r.matrix=this.matrix.toArray(),r.up=this.up.toArray(),this.pivot!==null&&(r.pivot=this.pivot.toArray()),this.matrixAutoUpdate===!1&&(r.matrixAutoUpdate=!1),this.morphTargetDictionary!==void 0&&(r.morphTargetDictionary=Object.assign({},this.morphTargetDictionary)),this.morphTargetInfluences!==void 0&&(r.morphTargetInfluences=this.morphTargetInfluences.slice()),this.isInstancedMesh&&(r.type="InstancedMesh",r.count=this.count,r.instanceMatrix=this.instanceMatrix.toJSON(),this.instanceColor!==null&&(r.instanceColor=this.instanceColor.toJSON())),this.isBatchedMesh&&(r.type="BatchedMesh",r.perObjectFrustumCulled=this.perObjectFrustumCulled,r.sortObjects=this.sortObjects,r.drawRanges=this._drawRanges,r.reservedRanges=this._reservedRanges,r.geometryInfo=this._geometryInfo.map(o=>({...o,boundingBox:o.boundingBox?o.boundingBox.toJSON():void 0,boundingSphere:o.boundingSphere?o.boundingSphere.toJSON():void 0})),r.instanceInfo=this._instanceInfo.map(o=>({...o})),r.availableInstanceIds=this._availableInstanceIds.slice(),r.availableGeometryIds=this._availableGeometryIds.slice(),r.nextIndexStart=this._nextIndexStart,r.nextVertexStart=this._nextVertexStart,r.geometryCount=this._geometryCount,r.maxInstanceCount=this._maxInstanceCount,r.maxVertexCount=this._maxVertexCount,r.maxIndexCount=this._maxIndexCount,r.geometryInitialized=this._geometryInitialized,r.matricesTexture=this._matricesTexture.toJSON(e),r.indirectTexture=this._indirectTexture.toJSON(e),this._colorsTexture!==null&&(r.colorsTexture=this._colorsTexture.toJSON(e)),this.boundingSphere!==null&&(r.boundingSphere=this.boundingSphere.toJSON()),this.boundingBox!==null&&(r.boundingBox=this.boundingBox.toJSON()));function s(o,l){return o[l.uuid]===void 0&&(o[l.uuid]=l.toJSON(e)),l.uuid}if(this.isScene)this.background&&(this.background.isColor?r.background=this.background.toJSON():this.background.isTexture&&(r.background=this.background.toJSON(e).uuid)),this.environment&&this.environment.isTexture&&this.environment.isRenderTargetTexture!==!0&&(r.environment=this.environment.toJSON(e).uuid);else if(this.isMesh||this.isLine||this.isPoints){r.geometry=s(e.geometries,this.geometry);const o=this.geometry.parameters;if(o!==void 0&&o.shapes!==void 0){const l=o.shapes;if(Array.isArray(l))for(let c=0,f=l.length;c<f;c++){const h=l[c];s(e.shapes,h)}else s(e.shapes,l)}}if(this.isSkinnedMesh&&(r.bindMode=this.bindMode,r.bindMatrix=this.bindMatrix.toArray(),this.skeleton!==void 0&&(s(e.skeletons,this.skeleton),r.skeleton=this.skeleton.uuid)),this.material!==void 0)if(Array.isArray(this.material)){const o=[];for(let l=0,c=this.material.length;l<c;l++)o.push(s(e.materials,this.material[l]));r.material=o}else r.material=s(e.materials,this.material);if(this.children.length>0){r.children=[];for(let o=0;o<this.children.length;o++)r.children.push(this.children[o].toJSON(e).object)}if(this.animations.length>0){r.animations=[];for(let o=0;o<this.animations.length;o++){const l=this.animations[o];r.animations.push(s(e.animations,l))}}if(t){const o=a(e.geometries),l=a(e.materials),c=a(e.textures),f=a(e.images),h=a(e.shapes),u=a(e.skeletons),d=a(e.animations),x=a(e.nodes);o.length>0&&(n.geometries=o),l.length>0&&(n.materials=l),c.length>0&&(n.textures=c),f.length>0&&(n.images=f),h.length>0&&(n.shapes=h),u.length>0&&(n.skeletons=u),d.length>0&&(n.animations=d),x.length>0&&(n.nodes=x)}return n.object=r,n;function a(o){const l=[];for(const c in o){const f=o[c];delete f.metadata,l.push(f)}return l}}clone(e){return new this.constructor().copy(this,e)}copy(e,t=!0){if(this.name=e.name,this.up.copy(e.up),this.position.copy(e.position),this.rotation.order=e.rotation.order,this.quaternion.copy(e.quaternion),this.scale.copy(e.scale),this.pivot=e.pivot!==null?e.pivot.clone():null,this.matrix.copy(e.matrix),this.matrixWorld.copy(e.matrixWorld),this.matrixAutoUpdate=e.matrixAutoUpdate,this.matrixWorldAutoUpdate=e.matrixWorldAutoUpdate,this.matrixWorldNeedsUpdate=e.matrixWorldNeedsUpdate,this.layers.mask=e.layers.mask,this.visible=e.visible,this.castShadow=e.castShadow,this.receiveShadow=e.receiveShadow,this.frustumCulled=e.frustumCulled,this.renderOrder=e.renderOrder,this.static=e.static,this.animations=e.animations.slice(),this.userData=JSON.parse(JSON.stringify(e.userData)),t===!0)for(let n=0;n<e.children.length;n++){const r=e.children[n];this.add(r.clone())}return this}}_n.DEFAULT_UP=new X(0,1,0);_n.DEFAULT_MATRIX_AUTO_UPDATE=!0;_n.DEFAULT_MATRIX_WORLD_AUTO_UPDATE=!0;class ha extends _n{constructor(){super(),this.isGroup=!0,this.type="Group"}}const k0={type:"move"};class pl{constructor(){this._targetRay=null,this._grip=null,this._hand=null}getHandSpace(){return this._hand===null&&(this._hand=new ha,this._hand.matrixAutoUpdate=!1,this._hand.visible=!1,this._hand.joints={},this._hand.inputState={pinching:!1}),this._hand}getTargetRaySpace(){return this._targetRay===null&&(this._targetRay=new ha,this._targetRay.matrixAutoUpdate=!1,this._targetRay.visible=!1,this._targetRay.hasLinearVelocity=!1,this._targetRay.linearVelocity=new X,this._targetRay.hasAngularVelocity=!1,this._targetRay.angularVelocity=new X),this._targetRay}getGripSpace(){return this._grip===null&&(this._grip=new ha,this._grip.matrixAutoUpdate=!1,this._grip.visible=!1,this._grip.hasLinearVelocity=!1,this._grip.linearVelocity=new X,this._grip.hasAngularVelocity=!1,this._grip.angularVelocity=new X,this._grip.eventsEnabled=!1),this._grip}dispatchEvent(e){return this._targetRay!==null&&this._targetRay.dispatchEvent(e),this._grip!==null&&this._grip.dispatchEvent(e),this._hand!==null&&this._hand.dispatchEvent(e),this}connect(e){if(e&&e.hand){const t=this._hand;if(t)for(const n of e.hand.values())this._getHandJoint(t,n)}return this.dispatchEvent({type:"connected",data:e}),this}disconnect(e){return this.dispatchEvent({type:"disconnected",data:e}),this._targetRay!==null&&(this._targetRay.visible=!1),this._grip!==null&&(this._grip.visible=!1),this._hand!==null&&(this._hand.visible=!1),this}update(e,t,n){let r=null,s=null,a=null;const o=this._targetRay,l=this._grip,c=this._hand;if(e&&t.session.visibilityState!=="visible-blurred"){if(c&&e.hand){a=!0;for(const E of e.hand.values()){const m=t.getJointPose(E,n),p=this._getHandJoint(c,E);m!==null&&(p.matrix.fromArray(m.transform.matrix),p.matrix.decompose(p.position,p.rotation,p.scale),p.matrixWorldNeedsUpdate=!0,p.jointRadius=m.radius),p.visible=m!==null}const f=c.joints["index-finger-tip"],h=c.joints["thumb-tip"],u=f.position.distanceTo(h.position),d=.02,x=.005;c.inputState.pinching&&u>d+x?(c.inputState.pinching=!1,this.dispatchEvent({type:"pinchend",handedness:e.handedness,target:this})):!c.inputState.pinching&&u<=d-x&&(c.inputState.pinching=!0,this.dispatchEvent({type:"pinchstart",handedness:e.handedness,target:this}))}else l!==null&&e.gripSpace&&(s=t.getPose(e.gripSpace,n),s!==null&&(l.matrix.fromArray(s.transform.matrix),l.matrix.decompose(l.position,l.rotation,l.scale),l.matrixWorldNeedsUpdate=!0,s.linearVelocity?(l.hasLinearVelocity=!0,l.linearVelocity.copy(s.linearVelocity)):l.hasLinearVelocity=!1,s.angularVelocity?(l.hasAngularVelocity=!0,l.angularVelocity.copy(s.angularVelocity)):l.hasAngularVelocity=!1,l.eventsEnabled&&l.dispatchEvent({type:"gripUpdated",data:e,target:this})));o!==null&&(r=t.getPose(e.targetRaySpace,n),r===null&&s!==null&&(r=s),r!==null&&(o.matrix.fromArray(r.transform.matrix),o.matrix.decompose(o.position,o.rotation,o.scale),o.matrixWorldNeedsUpdate=!0,r.linearVelocity?(o.hasLinearVelocity=!0,o.linearVelocity.copy(r.linearVelocity)):o.hasLinearVelocity=!1,r.angularVelocity?(o.hasAngularVelocity=!0,o.angularVelocity.copy(r.angularVelocity)):o.hasAngularVelocity=!1,this.dispatchEvent(k0)))}return o!==null&&(o.visible=r!==null),l!==null&&(l.visible=s!==null),c!==null&&(c.visible=a!==null),this}_getHandJoint(e,t){if(e.joints[t.jointName]===void 0){const n=new ha;n.matrixAutoUpdate=!1,n.visible=!1,e.joints[t.jointName]=n,e.add(n)}return e.joints[t.jointName]}}const $d={aliceblue:15792383,antiquewhite:16444375,aqua:65535,aquamarine:8388564,azure:15794175,beige:16119260,bisque:16770244,black:0,blanchedalmond:16772045,blue:255,blueviolet:9055202,brown:10824234,burlywood:14596231,cadetblue:6266528,chartreuse:8388352,chocolate:13789470,coral:16744272,cornflowerblue:6591981,cornsilk:16775388,crimson:14423100,cyan:65535,darkblue:139,darkcyan:35723,darkgoldenrod:12092939,darkgray:11119017,darkgreen:25600,darkgrey:11119017,darkkhaki:12433259,darkmagenta:9109643,darkolivegreen:5597999,darkorange:16747520,darkorchid:10040012,darkred:9109504,darksalmon:15308410,darkseagreen:9419919,darkslateblue:4734347,darkslategray:3100495,darkslategrey:3100495,darkturquoise:52945,darkviolet:9699539,deeppink:16716947,deepskyblue:49151,dimgray:6908265,dimgrey:6908265,dodgerblue:2003199,firebrick:11674146,floralwhite:16775920,forestgreen:2263842,fuchsia:16711935,gainsboro:14474460,ghostwhite:16316671,gold:16766720,goldenrod:14329120,gray:8421504,green:32768,greenyellow:11403055,grey:8421504,honeydew:15794160,hotpink:16738740,indianred:13458524,indigo:4915330,ivory:16777200,khaki:15787660,lavender:15132410,lavenderblush:16773365,lawngreen:8190976,lemonchiffon:16775885,lightblue:11393254,lightcoral:15761536,lightcyan:14745599,lightgoldenrodyellow:16448210,lightgray:13882323,lightgreen:9498256,lightgrey:13882323,lightpink:16758465,lightsalmon:16752762,lightseagreen:2142890,lightskyblue:8900346,lightslategray:7833753,lightslategrey:7833753,lightsteelblue:11584734,lightyellow:16777184,lime:65280,limegreen:3329330,linen:16445670,magenta:16711935,maroon:8388608,mediumaquamarine:6737322,mediumblue:205,mediumorchid:12211667,mediumpurple:9662683,mediumseagreen:3978097,mediumslateblue:8087790,mediumspringgreen:64154,mediumturquoise:4772300,mediumvioletred:13047173,midnightblue:1644912,mintcream:16121850,mistyrose:16770273,moccasin:16770229,navajowhite:16768685,navy:128,oldlace:16643558,olive:8421376,olivedrab:7048739,orange:16753920,orangered:16729344,orchid:14315734,palegoldenrod:15657130,palegreen:10025880,paleturquoise:11529966,palevioletred:14381203,papayawhip:16773077,peachpuff:16767673,peru:13468991,pink:16761035,plum:14524637,powderblue:11591910,purple:8388736,rebeccapurple:6697881,red:16711680,rosybrown:12357519,royalblue:4286945,saddlebrown:9127187,salmon:16416882,sandybrown:16032864,seagreen:3050327,seashell:16774638,sienna:10506797,silver:12632256,skyblue:8900331,slateblue:6970061,slategray:7372944,slategrey:7372944,snow:16775930,springgreen:65407,steelblue:4620980,tan:13808780,teal:32896,thistle:14204888,tomato:16737095,turquoise:4251856,violet:15631086,wheat:16113331,white:16777215,whitesmoke:16119285,yellow:16776960,yellowgreen:10145074},ar={h:0,s:0,l:0},Ua={h:0,s:0,l:0};function ml(i,e,t){return t<0&&(t+=1),t>1&&(t-=1),t<1/6?i+(e-i)*6*t:t<1/2?e:t<2/3?i+(e-i)*6*(2/3-t):i}class ft{constructor(e,t,n){return this.isColor=!0,this.r=1,this.g=1,this.b=1,this.set(e,t,n)}set(e,t,n){if(t===void 0&&n===void 0){const r=e;r&&r.isColor?this.copy(r):typeof r=="number"?this.setHex(r):typeof r=="string"&&this.setStyle(r)}else this.setRGB(e,t,n);return this}setScalar(e){return this.r=e,this.g=e,this.b=e,this}setHex(e,t=ai){return e=Math.floor(e),this.r=(e>>16&255)/255,this.g=(e>>8&255)/255,this.b=(e&255)/255,mt.colorSpaceToWorking(this,t),this}setRGB(e,t,n,r=mt.workingColorSpace){return this.r=e,this.g=t,this.b=n,mt.colorSpaceToWorking(this,r),this}setHSL(e,t,n,r=mt.workingColorSpace){if(e=A0(e,1),t=ut(t,0,1),n=ut(n,0,1),t===0)this.r=this.g=this.b=n;else{const s=n<=.5?n*(1+t):n+t-n*t,a=2*n-s;this.r=ml(a,s,e+1/3),this.g=ml(a,s,e),this.b=ml(a,s,e-1/3)}return mt.colorSpaceToWorking(this,r),this}setStyle(e,t=ai){function n(s){s!==void 0&&parseFloat(s)<1&&Je("Color: Alpha component of "+e+" will be ignored.")}let r;if(r=/^(\w+)\(([^\)]*)\)/.exec(e)){let s;const a=r[1],o=r[2];switch(a){case"rgb":case"rgba":if(s=/^\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(o))return n(s[4]),this.setRGB(Math.min(255,parseInt(s[1],10))/255,Math.min(255,parseInt(s[2],10))/255,Math.min(255,parseInt(s[3],10))/255,t);if(s=/^\s*(\d+)\%\s*,\s*(\d+)\%\s*,\s*(\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(o))return n(s[4]),this.setRGB(Math.min(100,parseInt(s[1],10))/100,Math.min(100,parseInt(s[2],10))/100,Math.min(100,parseInt(s[3],10))/100,t);break;case"hsl":case"hsla":if(s=/^\s*(\d*\.?\d+)\s*,\s*(\d*\.?\d+)\%\s*,\s*(\d*\.?\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(o))return n(s[4]),this.setHSL(parseFloat(s[1])/360,parseFloat(s[2])/100,parseFloat(s[3])/100,t);break;default:Je("Color: Unknown color model "+e)}}else if(r=/^\#([A-Fa-f\d]+)$/.exec(e)){const s=r[1],a=s.length;if(a===3)return this.setRGB(parseInt(s.charAt(0),16)/15,parseInt(s.charAt(1),16)/15,parseInt(s.charAt(2),16)/15,t);if(a===6)return this.setHex(parseInt(s,16),t);Je("Color: Invalid hex color "+e)}else if(e&&e.length>0)return this.setColorName(e,t);return this}setColorName(e,t=ai){const n=$d[e.toLowerCase()];return n!==void 0?this.setHex(n,t):Je("Color: Unknown color "+e),this}clone(){return new this.constructor(this.r,this.g,this.b)}copy(e){return this.r=e.r,this.g=e.g,this.b=e.b,this}copySRGBToLinear(e){return this.r=$i(e.r),this.g=$i(e.g),this.b=$i(e.b),this}copyLinearToSRGB(e){return this.r=As(e.r),this.g=As(e.g),this.b=As(e.b),this}convertSRGBToLinear(){return this.copySRGBToLinear(this),this}convertLinearToSRGB(){return this.copyLinearToSRGB(this),this}getHex(e=ai){return mt.workingToColorSpace(Tn.copy(this),e),Math.round(ut(Tn.r*255,0,255))*65536+Math.round(ut(Tn.g*255,0,255))*256+Math.round(ut(Tn.b*255,0,255))}getHexString(e=ai){return("000000"+this.getHex(e).toString(16)).slice(-6)}getHSL(e,t=mt.workingColorSpace){mt.workingToColorSpace(Tn.copy(this),t);const n=Tn.r,r=Tn.g,s=Tn.b,a=Math.max(n,r,s),o=Math.min(n,r,s);let l,c;const f=(o+a)/2;if(o===a)l=0,c=0;else{const h=a-o;switch(c=f<=.5?h/(a+o):h/(2-a-o),a){case n:l=(r-s)/h+(r<s?6:0);break;case r:l=(s-n)/h+2;break;case s:l=(n-r)/h+4;break}l/=6}return e.h=l,e.s=c,e.l=f,e}getRGB(e,t=mt.workingColorSpace){return mt.workingToColorSpace(Tn.copy(this),t),e.r=Tn.r,e.g=Tn.g,e.b=Tn.b,e}getStyle(e=ai){mt.workingToColorSpace(Tn.copy(this),e);const t=Tn.r,n=Tn.g,r=Tn.b;return e!==ai?`color(${e} ${t.toFixed(3)} ${n.toFixed(3)} ${r.toFixed(3)})`:`rgb(${Math.round(t*255)},${Math.round(n*255)},${Math.round(r*255)})`}offsetHSL(e,t,n){return this.getHSL(ar),this.setHSL(ar.h+e,ar.s+t,ar.l+n)}add(e){return this.r+=e.r,this.g+=e.g,this.b+=e.b,this}addColors(e,t){return this.r=e.r+t.r,this.g=e.g+t.g,this.b=e.b+t.b,this}addScalar(e){return this.r+=e,this.g+=e,this.b+=e,this}sub(e){return this.r=Math.max(0,this.r-e.r),this.g=Math.max(0,this.g-e.g),this.b=Math.max(0,this.b-e.b),this}multiply(e){return this.r*=e.r,this.g*=e.g,this.b*=e.b,this}multiplyScalar(e){return this.r*=e,this.g*=e,this.b*=e,this}lerp(e,t){return this.r+=(e.r-this.r)*t,this.g+=(e.g-this.g)*t,this.b+=(e.b-this.b)*t,this}lerpColors(e,t,n){return this.r=e.r+(t.r-e.r)*n,this.g=e.g+(t.g-e.g)*n,this.b=e.b+(t.b-e.b)*n,this}lerpHSL(e,t){this.getHSL(ar),e.getHSL(Ua);const n=ll(ar.h,Ua.h,t),r=ll(ar.s,Ua.s,t),s=ll(ar.l,Ua.l,t);return this.setHSL(n,r,s),this}setFromVector3(e){return this.r=e.x,this.g=e.y,this.b=e.z,this}applyMatrix3(e){const t=this.r,n=this.g,r=this.b,s=e.elements;return this.r=s[0]*t+s[3]*n+s[6]*r,this.g=s[1]*t+s[4]*n+s[7]*r,this.b=s[2]*t+s[5]*n+s[8]*r,this}equals(e){return e.r===this.r&&e.g===this.g&&e.b===this.b}fromArray(e,t=0){return this.r=e[t],this.g=e[t+1],this.b=e[t+2],this}toArray(e=[],t=0){return e[t]=this.r,e[t+1]=this.g,e[t+2]=this.b,e}fromBufferAttribute(e,t){return this.r=e.getX(t),this.g=e.getY(t),this.b=e.getZ(t),this}toJSON(){return this.getHex()}*[Symbol.iterator](){yield this.r,yield this.g,yield this.b}}const Tn=new ft;ft.NAMES=$d;class V0 extends _n{constructor(){super(),this.isScene=!0,this.type="Scene",this.background=null,this.environment=null,this.fog=null,this.backgroundBlurriness=0,this.backgroundIntensity=1,this.backgroundRotation=new Gr,this.environmentIntensity=1,this.environmentRotation=new Gr,this.overrideMaterial=null,typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}copy(e,t){return super.copy(e,t),e.background!==null&&(this.background=e.background.clone()),e.environment!==null&&(this.environment=e.environment.clone()),e.fog!==null&&(this.fog=e.fog.clone()),this.backgroundBlurriness=e.backgroundBlurriness,this.backgroundIntensity=e.backgroundIntensity,this.backgroundRotation.copy(e.backgroundRotation),this.environmentIntensity=e.environmentIntensity,this.environmentRotation.copy(e.environmentRotation),e.overrideMaterial!==null&&(this.overrideMaterial=e.overrideMaterial.clone()),this.matrixAutoUpdate=e.matrixAutoUpdate,this}toJSON(e){const t=super.toJSON(e);return this.fog!==null&&(t.object.fog=this.fog.toJSON()),this.backgroundBlurriness>0&&(t.object.backgroundBlurriness=this.backgroundBlurriness),this.backgroundIntensity!==1&&(t.object.backgroundIntensity=this.backgroundIntensity),t.object.backgroundRotation=this.backgroundRotation.toArray(),this.environmentIntensity!==1&&(t.object.environmentIntensity=this.environmentIntensity),t.object.environmentRotation=this.environmentRotation.toArray(),t}}const di=new X,Vi=new X,gl=new X,Gi=new X,os=new X,ls=new X,Jf=new X,_l=new X,xl=new X,vl=new X,Sl=new $t,Ml=new $t,yl=new $t;class li{constructor(e=new X,t=new X,n=new X){this.a=e,this.b=t,this.c=n}static getNormal(e,t,n,r){r.subVectors(n,t),di.subVectors(e,t),r.cross(di);const s=r.lengthSq();return s>0?r.multiplyScalar(1/Math.sqrt(s)):r.set(0,0,0)}static getBarycoord(e,t,n,r,s){di.subVectors(r,t),Vi.subVectors(n,t),gl.subVectors(e,t);const a=di.dot(di),o=di.dot(Vi),l=di.dot(gl),c=Vi.dot(Vi),f=Vi.dot(gl),h=a*c-o*o;if(h===0)return s.set(0,0,0),null;const u=1/h,d=(c*l-o*f)*u,x=(a*f-o*l)*u;return s.set(1-d-x,x,d)}static containsPoint(e,t,n,r){return this.getBarycoord(e,t,n,r,Gi)===null?!1:Gi.x>=0&&Gi.y>=0&&Gi.x+Gi.y<=1}static getInterpolation(e,t,n,r,s,a,o,l){return this.getBarycoord(e,t,n,r,Gi)===null?(l.x=0,l.y=0,"z"in l&&(l.z=0),"w"in l&&(l.w=0),null):(l.setScalar(0),l.addScaledVector(s,Gi.x),l.addScaledVector(a,Gi.y),l.addScaledVector(o,Gi.z),l)}static getInterpolatedAttribute(e,t,n,r,s,a){return Sl.setScalar(0),Ml.setScalar(0),yl.setScalar(0),Sl.fromBufferAttribute(e,t),Ml.fromBufferAttribute(e,n),yl.fromBufferAttribute(e,r),a.setScalar(0),a.addScaledVector(Sl,s.x),a.addScaledVector(Ml,s.y),a.addScaledVector(yl,s.z),a}static isFrontFacing(e,t,n,r){return di.subVectors(n,t),Vi.subVectors(e,t),di.cross(Vi).dot(r)<0}set(e,t,n){return this.a.copy(e),this.b.copy(t),this.c.copy(n),this}setFromPointsAndIndices(e,t,n,r){return this.a.copy(e[t]),this.b.copy(e[n]),this.c.copy(e[r]),this}setFromAttributeAndIndices(e,t,n,r){return this.a.fromBufferAttribute(e,t),this.b.fromBufferAttribute(e,n),this.c.fromBufferAttribute(e,r),this}clone(){return new this.constructor().copy(this)}copy(e){return this.a.copy(e.a),this.b.copy(e.b),this.c.copy(e.c),this}getArea(){return di.subVectors(this.c,this.b),Vi.subVectors(this.a,this.b),di.cross(Vi).length()*.5}getMidpoint(e){return e.addVectors(this.a,this.b).add(this.c).multiplyScalar(1/3)}getNormal(e){return li.getNormal(this.a,this.b,this.c,e)}getPlane(e){return e.setFromCoplanarPoints(this.a,this.b,this.c)}getBarycoord(e,t){return li.getBarycoord(e,this.a,this.b,this.c,t)}getInterpolation(e,t,n,r,s){return li.getInterpolation(e,this.a,this.b,this.c,t,n,r,s)}containsPoint(e){return li.containsPoint(e,this.a,this.b,this.c)}isFrontFacing(e){return li.isFrontFacing(this.a,this.b,this.c,e)}intersectsBox(e){return e.intersectsTriangle(this)}closestPointToPoint(e,t){const n=this.a,r=this.b,s=this.c;let a,o;os.subVectors(r,n),ls.subVectors(s,n),_l.subVectors(e,n);const l=os.dot(_l),c=ls.dot(_l);if(l<=0&&c<=0)return t.copy(n);xl.subVectors(e,r);const f=os.dot(xl),h=ls.dot(xl);if(f>=0&&h<=f)return t.copy(r);const u=l*h-f*c;if(u<=0&&l>=0&&f<=0)return a=l/(l-f),t.copy(n).addScaledVector(os,a);vl.subVectors(e,s);const d=os.dot(vl),x=ls.dot(vl);if(x>=0&&d<=x)return t.copy(s);const E=d*c-l*x;if(E<=0&&c>=0&&x<=0)return o=c/(c-x),t.copy(n).addScaledVector(ls,o);const m=f*x-d*h;if(m<=0&&h-f>=0&&d-x>=0)return Jf.subVectors(s,r),o=(h-f)/(h-f+(d-x)),t.copy(r).addScaledVector(Jf,o);const p=1/(m+E+u);return a=E*p,o=u*p,t.copy(n).addScaledVector(os,a).addScaledVector(ls,o)}equals(e){return e.a.equals(this.a)&&e.b.equals(this.b)&&e.c.equals(this.c)}}class Sa{constructor(e=new X(1/0,1/0,1/0),t=new X(-1/0,-1/0,-1/0)){this.isBox3=!0,this.min=e,this.max=t}set(e,t){return this.min.copy(e),this.max.copy(t),this}setFromArray(e){this.makeEmpty();for(let t=0,n=e.length;t<n;t+=3)this.expandByPoint(pi.fromArray(e,t));return this}setFromBufferAttribute(e){this.makeEmpty();for(let t=0,n=e.count;t<n;t++)this.expandByPoint(pi.fromBufferAttribute(e,t));return this}setFromPoints(e){this.makeEmpty();for(let t=0,n=e.length;t<n;t++)this.expandByPoint(e[t]);return this}setFromCenterAndSize(e,t){const n=pi.copy(t).multiplyScalar(.5);return this.min.copy(e).sub(n),this.max.copy(e).add(n),this}setFromObject(e,t=!1){return this.makeEmpty(),this.expandByObject(e,t)}clone(){return new this.constructor().copy(this)}copy(e){return this.min.copy(e.min),this.max.copy(e.max),this}makeEmpty(){return this.min.x=this.min.y=this.min.z=1/0,this.max.x=this.max.y=this.max.z=-1/0,this}isEmpty(){return this.max.x<this.min.x||this.max.y<this.min.y||this.max.z<this.min.z}getCenter(e){return this.isEmpty()?e.set(0,0,0):e.addVectors(this.min,this.max).multiplyScalar(.5)}getSize(e){return this.isEmpty()?e.set(0,0,0):e.subVectors(this.max,this.min)}expandByPoint(e){return this.min.min(e),this.max.max(e),this}expandByVector(e){return this.min.sub(e),this.max.add(e),this}expandByScalar(e){return this.min.addScalar(-e),this.max.addScalar(e),this}expandByObject(e,t=!1){e.updateWorldMatrix(!1,!1);const n=e.geometry;if(n!==void 0){const s=n.getAttribute("position");if(t===!0&&s!==void 0&&e.isInstancedMesh!==!0)for(let a=0,o=s.count;a<o;a++)e.isMesh===!0?e.getVertexPosition(a,pi):pi.fromBufferAttribute(s,a),pi.applyMatrix4(e.matrixWorld),this.expandByPoint(pi);else e.boundingBox!==void 0?(e.boundingBox===null&&e.computeBoundingBox(),Na.copy(e.boundingBox)):(n.boundingBox===null&&n.computeBoundingBox(),Na.copy(n.boundingBox)),Na.applyMatrix4(e.matrixWorld),this.union(Na)}const r=e.children;for(let s=0,a=r.length;s<a;s++)this.expandByObject(r[s],t);return this}containsPoint(e){return e.x>=this.min.x&&e.x<=this.max.x&&e.y>=this.min.y&&e.y<=this.max.y&&e.z>=this.min.z&&e.z<=this.max.z}containsBox(e){return this.min.x<=e.min.x&&e.max.x<=this.max.x&&this.min.y<=e.min.y&&e.max.y<=this.max.y&&this.min.z<=e.min.z&&e.max.z<=this.max.z}getParameter(e,t){return t.set((e.x-this.min.x)/(this.max.x-this.min.x),(e.y-this.min.y)/(this.max.y-this.min.y),(e.z-this.min.z)/(this.max.z-this.min.z))}intersectsBox(e){return e.max.x>=this.min.x&&e.min.x<=this.max.x&&e.max.y>=this.min.y&&e.min.y<=this.max.y&&e.max.z>=this.min.z&&e.min.z<=this.max.z}intersectsSphere(e){return this.clampPoint(e.center,pi),pi.distanceToSquared(e.center)<=e.radius*e.radius}intersectsPlane(e){let t,n;return e.normal.x>0?(t=e.normal.x*this.min.x,n=e.normal.x*this.max.x):(t=e.normal.x*this.max.x,n=e.normal.x*this.min.x),e.normal.y>0?(t+=e.normal.y*this.min.y,n+=e.normal.y*this.max.y):(t+=e.normal.y*this.max.y,n+=e.normal.y*this.min.y),e.normal.z>0?(t+=e.normal.z*this.min.z,n+=e.normal.z*this.max.z):(t+=e.normal.z*this.max.z,n+=e.normal.z*this.min.z),t<=-e.constant&&n>=-e.constant}intersectsTriangle(e){if(this.isEmpty())return!1;this.getCenter(Qs),Fa.subVectors(this.max,Qs),cs.subVectors(e.a,Qs),us.subVectors(e.b,Qs),fs.subVectors(e.c,Qs),or.subVectors(us,cs),lr.subVectors(fs,us),Ar.subVectors(cs,fs);let t=[0,-or.z,or.y,0,-lr.z,lr.y,0,-Ar.z,Ar.y,or.z,0,-or.x,lr.z,0,-lr.x,Ar.z,0,-Ar.x,-or.y,or.x,0,-lr.y,lr.x,0,-Ar.y,Ar.x,0];return!El(t,cs,us,fs,Fa)||(t=[1,0,0,0,1,0,0,0,1],!El(t,cs,us,fs,Fa))?!1:(Oa.crossVectors(or,lr),t=[Oa.x,Oa.y,Oa.z],El(t,cs,us,fs,Fa))}clampPoint(e,t){return t.copy(e).clamp(this.min,this.max)}distanceToPoint(e){return this.clampPoint(e,pi).distanceTo(e)}getBoundingSphere(e){return this.isEmpty()?e.makeEmpty():(this.getCenter(e.center),e.radius=this.getSize(pi).length()*.5),e}intersect(e){return this.min.max(e.min),this.max.min(e.max),this.isEmpty()&&this.makeEmpty(),this}union(e){return this.min.min(e.min),this.max.max(e.max),this}applyMatrix4(e){return this.isEmpty()?this:(Hi[0].set(this.min.x,this.min.y,this.min.z).applyMatrix4(e),Hi[1].set(this.min.x,this.min.y,this.max.z).applyMatrix4(e),Hi[2].set(this.min.x,this.max.y,this.min.z).applyMatrix4(e),Hi[3].set(this.min.x,this.max.y,this.max.z).applyMatrix4(e),Hi[4].set(this.max.x,this.min.y,this.min.z).applyMatrix4(e),Hi[5].set(this.max.x,this.min.y,this.max.z).applyMatrix4(e),Hi[6].set(this.max.x,this.max.y,this.min.z).applyMatrix4(e),Hi[7].set(this.max.x,this.max.y,this.max.z).applyMatrix4(e),this.setFromPoints(Hi),this)}translate(e){return this.min.add(e),this.max.add(e),this}equals(e){return e.min.equals(this.min)&&e.max.equals(this.max)}toJSON(){return{min:this.min.toArray(),max:this.max.toArray()}}fromJSON(e){return this.min.fromArray(e.min),this.max.fromArray(e.max),this}}const Hi=[new X,new X,new X,new X,new X,new X,new X,new X],pi=new X,Na=new Sa,cs=new X,us=new X,fs=new X,or=new X,lr=new X,Ar=new X,Qs=new X,Fa=new X,Oa=new X,wr=new X;function El(i,e,t,n,r){for(let s=0,a=i.length-3;s<=a;s+=3){wr.fromArray(i,s);const o=r.x*Math.abs(wr.x)+r.y*Math.abs(wr.y)+r.z*Math.abs(wr.z),l=e.dot(wr),c=t.dot(wr),f=n.dot(wr);if(Math.max(-Math.max(l,c,f),Math.min(l,c,f))>o)return!1}return!0}const tn=new X,Ba=new $e;let G0=0;class Qn extends Sr{constructor(e,t,n=!1){if(super(),Array.isArray(e))throw new TypeError("THREE.BufferAttribute: array should be a Typed Array.");this.isBufferAttribute=!0,Object.defineProperty(this,"id",{value:G0++}),this.name="",this.array=e,this.itemSize=t,this.count=e!==void 0?e.length/t:0,this.normalized=n,this.usage=kc,this.updateRanges=[],this.gpuType=Li,this.version=0}onUploadCallback(){}set needsUpdate(e){e===!0&&this.version++}setUsage(e){return this.usage=e,this}addUpdateRange(e,t){this.updateRanges.push({start:e,count:t})}clearUpdateRanges(){this.updateRanges.length=0}copy(e){return this.name=e.name,this.array=new e.array.constructor(e.array),this.itemSize=e.itemSize,this.count=e.count,this.normalized=e.normalized,this.usage=e.usage,this.gpuType=e.gpuType,this}copyAt(e,t,n){e*=this.itemSize,n*=t.itemSize;for(let r=0,s=this.itemSize;r<s;r++)this.array[e+r]=t.array[n+r];return this}copyArray(e){return this.array.set(e),this}applyMatrix3(e){if(this.itemSize===2)for(let t=0,n=this.count;t<n;t++)Ba.fromBufferAttribute(this,t),Ba.applyMatrix3(e),this.setXY(t,Ba.x,Ba.y);else if(this.itemSize===3)for(let t=0,n=this.count;t<n;t++)tn.fromBufferAttribute(this,t),tn.applyMatrix3(e),this.setXYZ(t,tn.x,tn.y,tn.z);return this}applyMatrix4(e){for(let t=0,n=this.count;t<n;t++)tn.fromBufferAttribute(this,t),tn.applyMatrix4(e),this.setXYZ(t,tn.x,tn.y,tn.z);return this}applyNormalMatrix(e){for(let t=0,n=this.count;t<n;t++)tn.fromBufferAttribute(this,t),tn.applyNormalMatrix(e),this.setXYZ(t,tn.x,tn.y,tn.z);return this}transformDirection(e){for(let t=0,n=this.count;t<n;t++)tn.fromBufferAttribute(this,t),tn.transformDirection(e),this.setXYZ(t,tn.x,tn.y,tn.z);return this}set(e,t=0){return this.array.set(e,t),this}getComponent(e,t){let n=this.array[e*this.itemSize+t];return this.normalized&&(n=Di(n,this.array)),n}setComponent(e,t,n){return this.normalized&&(n=Lt(n,this.array)),this.array[e*this.itemSize+t]=n,this}getX(e){let t=this.array[e*this.itemSize];return this.normalized&&(t=Di(t,this.array)),t}setX(e,t){return this.normalized&&(t=Lt(t,this.array)),this.array[e*this.itemSize]=t,this}getY(e){let t=this.array[e*this.itemSize+1];return this.normalized&&(t=Di(t,this.array)),t}setY(e,t){return this.normalized&&(t=Lt(t,this.array)),this.array[e*this.itemSize+1]=t,this}getZ(e){let t=this.array[e*this.itemSize+2];return this.normalized&&(t=Di(t,this.array)),t}setZ(e,t){return this.normalized&&(t=Lt(t,this.array)),this.array[e*this.itemSize+2]=t,this}getW(e){let t=this.array[e*this.itemSize+3];return this.normalized&&(t=Di(t,this.array)),t}setW(e,t){return this.normalized&&(t=Lt(t,this.array)),this.array[e*this.itemSize+3]=t,this}setXY(e,t,n){return e*=this.itemSize,this.normalized&&(t=Lt(t,this.array),n=Lt(n,this.array)),this.array[e+0]=t,this.array[e+1]=n,this}setXYZ(e,t,n,r){return e*=this.itemSize,this.normalized&&(t=Lt(t,this.array),n=Lt(n,this.array),r=Lt(r,this.array)),this.array[e+0]=t,this.array[e+1]=n,this.array[e+2]=r,this}setXYZW(e,t,n,r,s){return e*=this.itemSize,this.normalized&&(t=Lt(t,this.array),n=Lt(n,this.array),r=Lt(r,this.array),s=Lt(s,this.array)),this.array[e+0]=t,this.array[e+1]=n,this.array[e+2]=r,this.array[e+3]=s,this}onUpload(e){return this.onUploadCallback=e,this}clone(){return new this.constructor(this.array,this.itemSize).copy(this)}toJSON(){const e={itemSize:this.itemSize,type:this.array.constructor.name,array:Array.from(this.array),normalized:this.normalized};return this.name!==""&&(e.name=this.name),this.usage!==kc&&(e.usage=this.usage),e}dispose(){this.dispatchEvent({type:"dispose"})}}class Zd extends Qn{constructor(e,t,n){super(new Uint16Array(e),t,n)}}class Jd extends Qn{constructor(e,t,n){super(new Uint32Array(e),t,n)}}class Bn extends Qn{constructor(e,t,n){super(new Float32Array(e),t,n)}}const H0=new Sa,ea=new X,bl=new X;class Bo{constructor(e=new X,t=-1){this.isSphere=!0,this.center=e,this.radius=t}set(e,t){return this.center.copy(e),this.radius=t,this}setFromPoints(e,t){const n=this.center;t!==void 0?n.copy(t):H0.setFromPoints(e).getCenter(n);let r=0;for(let s=0,a=e.length;s<a;s++)r=Math.max(r,n.distanceToSquared(e[s]));return this.radius=Math.sqrt(r),this}copy(e){return this.center.copy(e.center),this.radius=e.radius,this}isEmpty(){return this.radius<0}makeEmpty(){return this.center.set(0,0,0),this.radius=-1,this}containsPoint(e){return e.distanceToSquared(this.center)<=this.radius*this.radius}distanceToPoint(e){return e.distanceTo(this.center)-this.radius}intersectsSphere(e){const t=this.radius+e.radius;return e.center.distanceToSquared(this.center)<=t*t}intersectsBox(e){return e.intersectsSphere(this)}intersectsPlane(e){return Math.abs(e.distanceToPoint(this.center))<=this.radius}clampPoint(e,t){const n=this.center.distanceToSquared(e);return t.copy(e),n>this.radius*this.radius&&(t.sub(this.center).normalize(),t.multiplyScalar(this.radius).add(this.center)),t}getBoundingBox(e){return this.isEmpty()?(e.makeEmpty(),e):(e.set(this.center,this.center),e.expandByScalar(this.radius),e)}applyMatrix4(e){return this.center.applyMatrix4(e),this.radius=this.radius*e.getMaxScaleOnAxis(),this}translate(e){return this.center.add(e),this}expandByPoint(e){if(this.isEmpty())return this.center.copy(e),this.radius=0,this;ea.subVectors(e,this.center);const t=ea.lengthSq();if(t>this.radius*this.radius){const n=Math.sqrt(t),r=(n-this.radius)*.5;this.center.addScaledVector(ea,r/n),this.radius+=r}return this}union(e){return e.isEmpty()?this:this.isEmpty()?(this.copy(e),this):(this.center.equals(e.center)===!0?this.radius=Math.max(this.radius,e.radius):(bl.subVectors(e.center,this.center).setLength(e.radius),this.expandByPoint(ea.copy(e.center).add(bl)),this.expandByPoint(ea.copy(e.center).sub(bl))),this)}equals(e){return e.center.equals(this.center)&&e.radius===this.radius}clone(){return new this.constructor().copy(this)}toJSON(){return{radius:this.radius,center:this.center.toArray()}}fromJSON(e){return this.radius=e.radius,this.center.fromArray(e.center),this}}let W0=0;const ii=new Xt,Tl=new _n,hs=new X,$n=new Sa,ta=new Sa,mn=new X;class gn extends Sr{constructor(){super(),this.isBufferGeometry=!0,Object.defineProperty(this,"id",{value:W0++}),this.uuid=gr(),this.name="",this.type="BufferGeometry",this.index=null,this.indirect=null,this.indirectOffset=0,this.attributes={},this.morphAttributes={},this.morphTargetsRelative=!1,this.groups=[],this.boundingBox=null,this.boundingSphere=null,this.drawRange={start:0,count:1/0},this.userData={},this._transformed=!1}getIndex(){return this.index}setIndex(e){return Array.isArray(e)?this.index=new(y0(e)?Jd:Zd)(e,1):this.index=e,this}setIndirect(e,t=0){return this.indirect=e,this.indirectOffset=t,this}getIndirect(){return this.indirect}getAttribute(e){return this.attributes[e]}setAttribute(e,t){return this.attributes[e]=t,this}deleteAttribute(e){return delete this.attributes[e],this}hasAttribute(e){return this.attributes[e]!==void 0}addGroup(e,t,n=0){this.groups.push({start:e,count:t,materialIndex:n})}clearGroups(){this.groups=[]}setDrawRange(e,t){this.drawRange.start=e,this.drawRange.count=t}applyMatrix4(e){const t=this.attributes.position;t!==void 0&&(t.applyMatrix4(e),t.needsUpdate=!0);const n=this.attributes.normal;if(n!==void 0){const s=new nt().getNormalMatrix(e);n.applyNormalMatrix(s),n.needsUpdate=!0}const r=this.attributes.tangent;return r!==void 0&&(r.transformDirection(e),r.needsUpdate=!0),this.boundingBox!==null&&this.computeBoundingBox(),this.boundingSphere!==null&&this.computeBoundingSphere(),this._transformed=!0,this}applyQuaternion(e){return ii.makeRotationFromQuaternion(e),this.applyMatrix4(ii),this}rotateX(e){return ii.makeRotationX(e),this.applyMatrix4(ii),this}rotateY(e){return ii.makeRotationY(e),this.applyMatrix4(ii),this}rotateZ(e){return ii.makeRotationZ(e),this.applyMatrix4(ii),this}translate(e,t,n){return ii.makeTranslation(e,t,n),this.applyMatrix4(ii),this}scale(e,t,n){return ii.makeScale(e,t,n),this.applyMatrix4(ii),this}lookAt(e){return Tl.lookAt(e),Tl.updateMatrix(),this.applyMatrix4(Tl.matrix),this}center(){return this.computeBoundingBox(),this.boundingBox.getCenter(hs).negate(),this.translate(hs.x,hs.y,hs.z),this}setFromPoints(e){const t=this.getAttribute("position");if(t===void 0){const n=[];for(let r=0,s=e.length;r<s;r++){const a=e[r];n.push(a.x,a.y,a.z||0)}this.setAttribute("position",new Bn(n,3))}else{const n=Math.min(e.length,t.count);for(let r=0;r<n;r++){const s=e[r];t.setXYZ(r,s.x,s.y,s.z||0)}e.length>t.count&&Je("BufferGeometry: Buffer size too small for points data. Use .dispose() and create a new geometry."),t.needsUpdate=!0}return this}computeBoundingBox(){this.boundingBox===null&&(this.boundingBox=new Sa);const e=this.attributes.position,t=this.morphAttributes.position;if(e&&e.isGLBufferAttribute){gt("BufferGeometry.computeBoundingBox(): GLBufferAttribute requires a manual bounding box.",this),this.boundingBox.set(new X(-1/0,-1/0,-1/0),new X(1/0,1/0,1/0));return}if(e!==void 0){if(this.boundingBox.setFromBufferAttribute(e),t)for(let n=0,r=t.length;n<r;n++){const s=t[n];$n.setFromBufferAttribute(s),this.morphTargetsRelative?(mn.addVectors(this.boundingBox.min,$n.min),this.boundingBox.expandByPoint(mn),mn.addVectors(this.boundingBox.max,$n.max),this.boundingBox.expandByPoint(mn)):(this.boundingBox.expandByPoint($n.min),this.boundingBox.expandByPoint($n.max))}}else this.boundingBox.makeEmpty();(isNaN(this.boundingBox.min.x)||isNaN(this.boundingBox.min.y)||isNaN(this.boundingBox.min.z))&&gt('BufferGeometry.computeBoundingBox(): Computed min/max have NaN values. The "position" attribute is likely to have NaN values.',this)}computeBoundingSphere(){this.boundingSphere===null&&(this.boundingSphere=new Bo);const e=this.attributes.position,t=this.morphAttributes.position;if(e&&e.isGLBufferAttribute){gt("BufferGeometry.computeBoundingSphere(): GLBufferAttribute requires a manual bounding sphere.",this),this.boundingSphere.set(new X,1/0);return}if(e){const n=this.boundingSphere.center;if($n.setFromBufferAttribute(e),t)for(let s=0,a=t.length;s<a;s++){const o=t[s];ta.setFromBufferAttribute(o),this.morphTargetsRelative?(mn.addVectors($n.min,ta.min),$n.expandByPoint(mn),mn.addVectors($n.max,ta.max),$n.expandByPoint(mn)):($n.expandByPoint(ta.min),$n.expandByPoint(ta.max))}$n.getCenter(n);let r=0;for(let s=0,a=e.count;s<a;s++)mn.fromBufferAttribute(e,s),r=Math.max(r,n.distanceToSquared(mn));if(t)for(let s=0,a=t.length;s<a;s++){const o=t[s],l=this.morphTargetsRelative;for(let c=0,f=o.count;c<f;c++)mn.fromBufferAttribute(o,c),l&&(hs.fromBufferAttribute(e,c),mn.add(hs)),r=Math.max(r,n.distanceToSquared(mn))}this.boundingSphere.radius=Math.sqrt(r),isNaN(this.boundingSphere.radius)&&gt('BufferGeometry.computeBoundingSphere(): Computed radius is NaN. The "position" attribute is likely to have NaN values.',this)}}computeTangents(){const e=this.index,t=this.attributes;if(e===null||t.position===void 0||t.normal===void 0||t.uv===void 0){gt("BufferGeometry: .computeTangents() failed. Missing required attributes (index, position, normal or uv)");return}const n=t.position,r=t.normal,s=t.uv;let a=this.getAttribute("tangent");(a===void 0||a.count!==n.count)&&(a=new Qn(new Float32Array(4*n.count),4),this.setAttribute("tangent",a));const o=[],l=[];for(let v=0;v<n.count;v++)o[v]=new X,l[v]=new X;const c=new X,f=new X,h=new X,u=new $e,d=new $e,x=new $e,E=new X,m=new X;function p(v,A,F){c.fromBufferAttribute(n,v),f.fromBufferAttribute(n,A),h.fromBufferAttribute(n,F),u.fromBufferAttribute(s,v),d.fromBufferAttribute(s,A),x.fromBufferAttribute(s,F),f.sub(c),h.sub(c),d.sub(u),x.sub(u);const D=1/(d.x*x.y-x.x*d.y);isFinite(D)&&(E.copy(f).multiplyScalar(x.y).addScaledVector(h,-d.y).multiplyScalar(D),m.copy(h).multiplyScalar(d.x).addScaledVector(f,-x.x).multiplyScalar(D),o[v].add(E),o[A].add(E),o[F].add(E),l[v].add(m),l[A].add(m),l[F].add(m))}let T=this.groups;T.length===0&&(T=[{start:0,count:e.count}]);for(let v=0,A=T.length;v<A;++v){const F=T[v],D=F.start,z=F.count;for(let O=D,k=D+z;O<k;O+=3)p(e.getX(O+0),e.getX(O+1),e.getX(O+2))}const P=new X,S=new X,C=new X,y=new X;function I(v){C.fromBufferAttribute(r,v),y.copy(C);const A=o[v];P.copy(A),P.sub(C.multiplyScalar(C.dot(A))).normalize(),S.crossVectors(y,A);const D=S.dot(l[v])<0?-1:1;a.setXYZW(v,P.x,P.y,P.z,D)}for(let v=0,A=T.length;v<A;++v){const F=T[v],D=F.start,z=F.count;for(let O=D,k=D+z;O<k;O+=3)I(e.getX(O+0)),I(e.getX(O+1)),I(e.getX(O+2))}this._transformed=!0}computeVertexNormals(){const e=this.index,t=this.getAttribute("position");if(t!==void 0){let n=this.getAttribute("normal");if(n===void 0||n.count!==t.count)n=new Qn(new Float32Array(t.count*3),3),this.setAttribute("normal",n);else for(let u=0,d=n.count;u<d;u++)n.setXYZ(u,0,0,0);const r=new X,s=new X,a=new X,o=new X,l=new X,c=new X,f=new X,h=new X;if(e)for(let u=0,d=e.count;u<d;u+=3){const x=e.getX(u+0),E=e.getX(u+1),m=e.getX(u+2);r.fromBufferAttribute(t,x),s.fromBufferAttribute(t,E),a.fromBufferAttribute(t,m),f.subVectors(a,s),h.subVectors(r,s),f.cross(h),o.fromBufferAttribute(n,x),l.fromBufferAttribute(n,E),c.fromBufferAttribute(n,m),o.add(f),l.add(f),c.add(f),n.setXYZ(x,o.x,o.y,o.z),n.setXYZ(E,l.x,l.y,l.z),n.setXYZ(m,c.x,c.y,c.z)}else for(let u=0,d=t.count;u<d;u+=3)r.fromBufferAttribute(t,u+0),s.fromBufferAttribute(t,u+1),a.fromBufferAttribute(t,u+2),f.subVectors(a,s),h.subVectors(r,s),f.cross(h),n.setXYZ(u+0,f.x,f.y,f.z),n.setXYZ(u+1,f.x,f.y,f.z),n.setXYZ(u+2,f.x,f.y,f.z);this.normalizeNormals(),n.needsUpdate=!0}}normalizeNormals(){const e=this.attributes.normal;for(let t=0,n=e.count;t<n;t++)mn.fromBufferAttribute(e,t),mn.normalize(),e.setXYZ(t,mn.x,mn.y,mn.z)}toNonIndexed(){function e(o,l){const c=o.array,f=o.itemSize,h=o.normalized,u=new c.constructor(l.length*f);let d=0,x=0;for(let E=0,m=l.length;E<m;E++){o.isInterleavedBufferAttribute?d=l[E]*o.data.stride+o.offset:d=l[E]*f;for(let p=0;p<f;p++)u[x++]=c[d++]}return new Qn(u,f,h)}if(this.index===null)return Je("BufferGeometry.toNonIndexed(): BufferGeometry is already non-indexed."),this;const t=new gn,n=this.index.array,r=this.attributes;for(const o in r){const l=r[o],c=e(l,n);t.setAttribute(o,c)}const s=this.morphAttributes;for(const o in s){const l=[],c=s[o];for(let f=0,h=c.length;f<h;f++){const u=c[f],d=e(u,n);l.push(d)}t.morphAttributes[o]=l}t.morphTargetsRelative=this.morphTargetsRelative;const a=this.groups;for(let o=0,l=a.length;o<l;o++){const c=a[o];t.addGroup(c.start,c.count,c.materialIndex)}return t}toJSON(){const e={metadata:{version:4.7,type:"BufferGeometry",generator:"BufferGeometry.toJSON"}};if(e.uuid=this.uuid,e.type=this.parameters!==void 0&&this._transformed===!0?"BufferGeometry":this.type,this.name!==""&&(e.name=this.name),Object.keys(this.userData).length>0&&(e.userData=this.userData),this.parameters!==void 0&&this._transformed!==!0){const l=this.parameters;for(const c in l)l[c]!==void 0&&(e[c]=l[c]);return e}e.data={attributes:{}};const t=this.index;t!==null&&(e.data.index={type:t.array.constructor.name,array:Array.prototype.slice.call(t.array)});const n=this.attributes;for(const l in n){const c=n[l];e.data.attributes[l]=c.toJSON(e.data)}const r={};let s=!1;for(const l in this.morphAttributes){const c=this.morphAttributes[l],f=[];for(let h=0,u=c.length;h<u;h++){const d=c[h];f.push(d.toJSON(e.data))}f.length>0&&(r[l]=f,s=!0)}s&&(e.data.morphAttributes=r,e.data.morphTargetsRelative=this.morphTargetsRelative);const a=this.groups;a.length>0&&(e.data.groups=JSON.parse(JSON.stringify(a)));const o=this.boundingSphere;return o!==null&&(e.data.boundingSphere=o.toJSON()),e}clone(){return new this.constructor().copy(this)}copy(e){this.index=null,this.attributes={},this.morphAttributes={},this.groups=[],this.boundingBox=null,this.boundingSphere=null;const t={};this.name=e.name;const n=e.index;n!==null&&this.setIndex(n.clone());const r=e.attributes;for(const c in r){const f=r[c];this.setAttribute(c,f.clone(t))}const s=e.morphAttributes;for(const c in s){const f=[],h=s[c];for(let u=0,d=h.length;u<d;u++)f.push(h[u].clone(t));this.morphAttributes[c]=f}this.morphTargetsRelative=e.morphTargetsRelative;const a=e.groups;for(let c=0,f=a.length;c<f;c++){const h=a[c];this.addGroup(h.start,h.count,h.materialIndex)}const o=e.boundingBox;o!==null&&(this.boundingBox=o.clone());const l=e.boundingSphere;return l!==null&&(this.boundingSphere=l.clone()),this.drawRange.start=e.drawRange.start,this.drawRange.count=e.drawRange.count,this.userData=e.userData,this._transformed=e._transformed,this}dispose(){this.dispatchEvent({type:"dispose"})}}class X0{constructor(e,t){this.isInterleavedBuffer=!0,this.array=e,this.stride=t,this.count=e!==void 0?e.length/t:0,this.usage=kc,this.updateRanges=[],this.version=0,this.uuid=gr()}onUploadCallback(){}set needsUpdate(e){e===!0&&this.version++}setUsage(e){return this.usage=e,this}addUpdateRange(e,t){this.updateRanges.push({start:e,count:t})}clearUpdateRanges(){this.updateRanges.length=0}copy(e){return this.array=new e.array.constructor(e.array),this.count=e.count,this.stride=e.stride,this.usage=e.usage,this}copyAt(e,t,n){e*=this.stride,n*=t.stride;for(let r=0,s=this.stride;r<s;r++)this.array[e+r]=t.array[n+r];return this}set(e,t=0){return this.array.set(e,t),this}clone(e){e.arrayBuffers===void 0&&(e.arrayBuffers={}),this.array.buffer._uuid===void 0&&(this.array.buffer._uuid=gr()),e.arrayBuffers[this.array.buffer._uuid]===void 0&&(e.arrayBuffers[this.array.buffer._uuid]=this.array.slice(0).buffer);const t=new this.array.constructor(e.arrayBuffers[this.array.buffer._uuid]),n=new this.constructor(t,this.stride);return n.setUsage(this.usage),n}onUpload(e){return this.onUploadCallback=e,this}toJSON(e){return e.arrayBuffers===void 0&&(e.arrayBuffers={}),this.array.buffer._uuid===void 0&&(this.array.buffer._uuid=gr()),e.arrayBuffers[this.array.buffer._uuid]===void 0&&(e.arrayBuffers[this.array.buffer._uuid]=Array.from(new Uint32Array(this.array.buffer))),{uuid:this.uuid,buffer:this.array.buffer._uuid,type:this.array.constructor.name,stride:this.stride}}}const Nn=new X;class To{constructor(e,t,n,r=!1){this.isInterleavedBufferAttribute=!0,this.name="",this.data=e,this.itemSize=t,this.offset=n,this.normalized=r}get count(){return this.data.count}get array(){return this.data.array}set needsUpdate(e){this.data.needsUpdate=e}applyMatrix4(e){for(let t=0,n=this.data.count;t<n;t++)Nn.fromBufferAttribute(this,t),Nn.applyMatrix4(e),this.setXYZ(t,Nn.x,Nn.y,Nn.z);return this}applyNormalMatrix(e){for(let t=0,n=this.count;t<n;t++)Nn.fromBufferAttribute(this,t),Nn.applyNormalMatrix(e),this.setXYZ(t,Nn.x,Nn.y,Nn.z);return this}transformDirection(e){for(let t=0,n=this.count;t<n;t++)Nn.fromBufferAttribute(this,t),Nn.transformDirection(e),this.setXYZ(t,Nn.x,Nn.y,Nn.z);return this}getComponent(e,t){let n=this.array[e*this.data.stride+this.offset+t];return this.normalized&&(n=Di(n,this.array)),n}setComponent(e,t,n){return this.normalized&&(n=Lt(n,this.array)),this.data.array[e*this.data.stride+this.offset+t]=n,this}setX(e,t){return this.normalized&&(t=Lt(t,this.array)),this.data.array[e*this.data.stride+this.offset]=t,this}setY(e,t){return this.normalized&&(t=Lt(t,this.array)),this.data.array[e*this.data.stride+this.offset+1]=t,this}setZ(e,t){return this.normalized&&(t=Lt(t,this.array)),this.data.array[e*this.data.stride+this.offset+2]=t,this}setW(e,t){return this.normalized&&(t=Lt(t,this.array)),this.data.array[e*this.data.stride+this.offset+3]=t,this}getX(e){let t=this.data.array[e*this.data.stride+this.offset];return this.normalized&&(t=Di(t,this.array)),t}getY(e){let t=this.data.array[e*this.data.stride+this.offset+1];return this.normalized&&(t=Di(t,this.array)),t}getZ(e){let t=this.data.array[e*this.data.stride+this.offset+2];return this.normalized&&(t=Di(t,this.array)),t}getW(e){let t=this.data.array[e*this.data.stride+this.offset+3];return this.normalized&&(t=Di(t,this.array)),t}setXY(e,t,n){return e=e*this.data.stride+this.offset,this.normalized&&(t=Lt(t,this.array),n=Lt(n,this.array)),this.data.array[e+0]=t,this.data.array[e+1]=n,this}setXYZ(e,t,n,r){return e=e*this.data.stride+this.offset,this.normalized&&(t=Lt(t,this.array),n=Lt(n,this.array),r=Lt(r,this.array)),this.data.array[e+0]=t,this.data.array[e+1]=n,this.data.array[e+2]=r,this}setXYZW(e,t,n,r,s){return e=e*this.data.stride+this.offset,this.normalized&&(t=Lt(t,this.array),n=Lt(n,this.array),r=Lt(r,this.array),s=Lt(s,this.array)),this.data.array[e+0]=t,this.data.array[e+1]=n,this.data.array[e+2]=r,this.data.array[e+3]=s,this}clone(e){if(e===void 0){bo("InterleavedBufferAttribute.clone(): Cloning an interleaved buffer attribute will de-interleave buffer data.");const t=[];for(let n=0;n<this.count;n++){const r=n*this.data.stride+this.offset;for(let s=0;s<this.itemSize;s++)t.push(this.data.array[r+s])}return new Qn(new this.array.constructor(t),this.itemSize,this.normalized)}else return e.interleavedBuffers===void 0&&(e.interleavedBuffers={}),e.interleavedBuffers[this.data.uuid]===void 0&&(e.interleavedBuffers[this.data.uuid]=this.data.clone(e)),new To(e.interleavedBuffers[this.data.uuid],this.itemSize,this.offset,this.normalized)}toJSON(e){if(e===void 0){bo("InterleavedBufferAttribute.toJSON(): Serializing an interleaved buffer attribute will de-interleave buffer data.");const t=[];for(let n=0;n<this.count;n++){const r=n*this.data.stride+this.offset;for(let s=0;s<this.itemSize;s++)t.push(this.data.array[r+s])}return{itemSize:this.itemSize,type:this.array.constructor.name,array:t,normalized:this.normalized}}else return e.interleavedBuffers===void 0&&(e.interleavedBuffers={}),e.interleavedBuffers[this.data.uuid]===void 0&&(e.interleavedBuffers[this.data.uuid]=this.data.toJSON(e)),{isInterleavedBufferAttribute:!0,itemSize:this.itemSize,data:this.data.uuid,offset:this.offset,normalized:this.normalized}}}let Y0=0;class Bs extends Sr{constructor(){super(),this.isMaterial=!0,Object.defineProperty(this,"id",{value:Y0++}),this.uuid=gr(),this.name="",this.type="Material",this.blending=bs,this.side=xr,this.vertexColors=!1,this.opacity=1,this.transparent=!1,this.alphaHash=!1,this.blendSrc=Ql,this.blendDst=ec,this.blendEquation=Ir,this.blendSrcAlpha=null,this.blendDstAlpha=null,this.blendEquationAlpha=null,this.blendColor=new ft(0,0,0),this.blendAlpha=0,this.depthFunc=Ds,this.depthTest=!0,this.depthWrite=!0,this.stencilWriteMask=255,this.stencilFunc=Bf,this.stencilRef=0,this.stencilFuncMask=255,this.stencilFail=ns,this.stencilZFail=ns,this.stencilZPass=ns,this.stencilWrite=!1,this.clippingPlanes=null,this.clipIntersection=!1,this.clipShadows=!1,this.shadowSide=null,this.colorWrite=!0,this.precision=null,this.polygonOffset=!1,this.polygonOffsetFactor=0,this.polygonOffsetUnits=0,this.dithering=!1,this.alphaToCoverage=!1,this.premultipliedAlpha=!1,this.forceSinglePass=!1,this.allowOverride=!0,this.visible=!0,this.toneMapped=!0,this.userData={},this.version=0,this._alphaTest=0}get alphaTest(){return this._alphaTest}set alphaTest(e){this._alphaTest>0!=e>0&&this.version++,this._alphaTest=e}onBeforeRender(){}onBeforeCompile(){}customProgramCacheKey(){return this.onBeforeCompile.toString()}setValues(e){if(e!==void 0)for(const t in e){const n=e[t];if(n===void 0){Je(`Material: parameter '${t}' has value of undefined.`);continue}const r=this[t];if(r===void 0){Je(`Material: '${t}' is not a property of THREE.${this.type}.`);continue}r&&r.isColor?r.set(n):r&&r.isVector2&&n&&n.isVector2||r&&r.isEuler&&n&&n.isEuler||r&&r.isVector3&&n&&n.isVector3?r.copy(n):this[t]=n}}toJSON(e){const t=e===void 0||typeof e=="string";t&&(e={textures:{},images:{}});const n={metadata:{version:4.7,type:"Material",generator:"Material.toJSON"}};n.uuid=this.uuid,n.type=this.type,this.name!==""&&(n.name=this.name),this.color&&this.color.isColor&&(n.color=this.color.getHex()),this.roughness!==void 0&&(n.roughness=this.roughness),this.metalness!==void 0&&(n.metalness=this.metalness),this.sheen!==void 0&&(n.sheen=this.sheen),this.sheenColor&&this.sheenColor.isColor&&(n.sheenColor=this.sheenColor.getHex()),this.sheenRoughness!==void 0&&(n.sheenRoughness=this.sheenRoughness),this.emissive&&this.emissive.isColor&&(n.emissive=this.emissive.getHex()),this.emissiveIntensity!==void 0&&this.emissiveIntensity!==1&&(n.emissiveIntensity=this.emissiveIntensity),this.specular&&this.specular.isColor&&(n.specular=this.specular.getHex()),this.specularIntensity!==void 0&&(n.specularIntensity=this.specularIntensity),this.specularColor&&this.specularColor.isColor&&(n.specularColor=this.specularColor.getHex()),this.shininess!==void 0&&(n.shininess=this.shininess),this.clearcoat!==void 0&&(n.clearcoat=this.clearcoat),this.clearcoatRoughness!==void 0&&(n.clearcoatRoughness=this.clearcoatRoughness),this.clearcoatMap&&this.clearcoatMap.isTexture&&(n.clearcoatMap=this.clearcoatMap.toJSON(e).uuid),this.clearcoatRoughnessMap&&this.clearcoatRoughnessMap.isTexture&&(n.clearcoatRoughnessMap=this.clearcoatRoughnessMap.toJSON(e).uuid),this.clearcoatNormalMap&&this.clearcoatNormalMap.isTexture&&(n.clearcoatNormalMap=this.clearcoatNormalMap.toJSON(e).uuid,n.clearcoatNormalScale=this.clearcoatNormalScale.toArray()),this.sheenColorMap&&this.sheenColorMap.isTexture&&(n.sheenColorMap=this.sheenColorMap.toJSON(e).uuid),this.sheenRoughnessMap&&this.sheenRoughnessMap.isTexture&&(n.sheenRoughnessMap=this.sheenRoughnessMap.toJSON(e).uuid),this.dispersion!==void 0&&(n.dispersion=this.dispersion),this.iridescence!==void 0&&(n.iridescence=this.iridescence),this.iridescenceIOR!==void 0&&(n.iridescenceIOR=this.iridescenceIOR),this.iridescenceThicknessRange!==void 0&&(n.iridescenceThicknessRange=this.iridescenceThicknessRange),this.iridescenceMap&&this.iridescenceMap.isTexture&&(n.iridescenceMap=this.iridescenceMap.toJSON(e).uuid),this.iridescenceThicknessMap&&this.iridescenceThicknessMap.isTexture&&(n.iridescenceThicknessMap=this.iridescenceThicknessMap.toJSON(e).uuid),this.anisotropy!==void 0&&(n.anisotropy=this.anisotropy),this.anisotropyRotation!==void 0&&(n.anisotropyRotation=this.anisotropyRotation),this.anisotropyMap&&this.anisotropyMap.isTexture&&(n.anisotropyMap=this.anisotropyMap.toJSON(e).uuid),this.map&&this.map.isTexture&&(n.map=this.map.toJSON(e).uuid),this.matcap&&this.matcap.isTexture&&(n.matcap=this.matcap.toJSON(e).uuid),this.alphaMap&&this.alphaMap.isTexture&&(n.alphaMap=this.alphaMap.toJSON(e).uuid),this.lightMap&&this.lightMap.isTexture&&(n.lightMap=this.lightMap.toJSON(e).uuid,n.lightMapIntensity=this.lightMapIntensity),this.aoMap&&this.aoMap.isTexture&&(n.aoMap=this.aoMap.toJSON(e).uuid,n.aoMapIntensity=this.aoMapIntensity),this.bumpMap&&this.bumpMap.isTexture&&(n.bumpMap=this.bumpMap.toJSON(e).uuid,n.bumpScale=this.bumpScale),this.normalMap&&this.normalMap.isTexture&&(n.normalMap=this.normalMap.toJSON(e).uuid,n.normalMapType=this.normalMapType,n.normalScale=this.normalScale.toArray()),this.displacementMap&&this.displacementMap.isTexture&&(n.displacementMap=this.displacementMap.toJSON(e).uuid,n.displacementScale=this.displacementScale,n.displacementBias=this.displacementBias),this.roughnessMap&&this.roughnessMap.isTexture&&(n.roughnessMap=this.roughnessMap.toJSON(e).uuid),this.metalnessMap&&this.metalnessMap.isTexture&&(n.metalnessMap=this.metalnessMap.toJSON(e).uuid),this.emissiveMap&&this.emissiveMap.isTexture&&(n.emissiveMap=this.emissiveMap.toJSON(e).uuid),this.specularMap&&this.specularMap.isTexture&&(n.specularMap=this.specularMap.toJSON(e).uuid),this.specularIntensityMap&&this.specularIntensityMap.isTexture&&(n.specularIntensityMap=this.specularIntensityMap.toJSON(e).uuid),this.specularColorMap&&this.specularColorMap.isTexture&&(n.specularColorMap=this.specularColorMap.toJSON(e).uuid),this.envMap&&this.envMap.isTexture&&(n.envMap=this.envMap.toJSON(e).uuid,this.combine!==void 0&&(n.combine=this.combine)),this.envMapRotation!==void 0&&(n.envMapRotation=this.envMapRotation.toArray()),this.envMapIntensity!==void 0&&(n.envMapIntensity=this.envMapIntensity),this.reflectivity!==void 0&&(n.reflectivity=this.reflectivity),this.refractionRatio!==void 0&&(n.refractionRatio=this.refractionRatio),this.gradientMap&&this.gradientMap.isTexture&&(n.gradientMap=this.gradientMap.toJSON(e).uuid),this.transmission!==void 0&&(n.transmission=this.transmission),this.transmissionMap&&this.transmissionMap.isTexture&&(n.transmissionMap=this.transmissionMap.toJSON(e).uuid),this.thickness!==void 0&&(n.thickness=this.thickness),this.thicknessMap&&this.thicknessMap.isTexture&&(n.thicknessMap=this.thicknessMap.toJSON(e).uuid),this.attenuationDistance!==void 0&&this.attenuationDistance!==1/0&&(n.attenuationDistance=this.attenuationDistance),this.attenuationColor!==void 0&&(n.attenuationColor=this.attenuationColor.getHex()),this.size!==void 0&&(n.size=this.size),this.shadowSide!==null&&(n.shadowSide=this.shadowSide),this.sizeAttenuation!==void 0&&(n.sizeAttenuation=this.sizeAttenuation),this.blending!==bs&&(n.blending=this.blending),this.side!==xr&&(n.side=this.side),this.vertexColors===!0&&(n.vertexColors=!0),this.opacity<1&&(n.opacity=this.opacity),this.transparent===!0&&(n.transparent=!0),this.blendSrc!==Ql&&(n.blendSrc=this.blendSrc),this.blendDst!==ec&&(n.blendDst=this.blendDst),this.blendEquation!==Ir&&(n.blendEquation=this.blendEquation),this.blendSrcAlpha!==null&&(n.blendSrcAlpha=this.blendSrcAlpha),this.blendDstAlpha!==null&&(n.blendDstAlpha=this.blendDstAlpha),this.blendEquationAlpha!==null&&(n.blendEquationAlpha=this.blendEquationAlpha),this.blendColor&&this.blendColor.isColor&&(n.blendColor=this.blendColor.getHex()),this.blendAlpha!==0&&(n.blendAlpha=this.blendAlpha),this.depthFunc!==Ds&&(n.depthFunc=this.depthFunc),this.depthTest===!1&&(n.depthTest=this.depthTest),this.depthWrite===!1&&(n.depthWrite=this.depthWrite),this.colorWrite===!1&&(n.colorWrite=this.colorWrite),this.stencilWriteMask!==255&&(n.stencilWriteMask=this.stencilWriteMask),this.stencilFunc!==Bf&&(n.stencilFunc=this.stencilFunc),this.stencilRef!==0&&(n.stencilRef=this.stencilRef),this.stencilFuncMask!==255&&(n.stencilFuncMask=this.stencilFuncMask),this.stencilFail!==ns&&(n.stencilFail=this.stencilFail),this.stencilZFail!==ns&&(n.stencilZFail=this.stencilZFail),this.stencilZPass!==ns&&(n.stencilZPass=this.stencilZPass),this.stencilWrite===!0&&(n.stencilWrite=this.stencilWrite),this.rotation!==void 0&&this.rotation!==0&&(n.rotation=this.rotation),this.polygonOffset===!0&&(n.polygonOffset=!0),this.polygonOffsetFactor!==0&&(n.polygonOffsetFactor=this.polygonOffsetFactor),this.polygonOffsetUnits!==0&&(n.polygonOffsetUnits=this.polygonOffsetUnits),this.linewidth!==void 0&&this.linewidth!==1&&(n.linewidth=this.linewidth),this.dashSize!==void 0&&(n.dashSize=this.dashSize),this.gapSize!==void 0&&(n.gapSize=this.gapSize),this.scale!==void 0&&(n.scale=this.scale),this.dithering===!0&&(n.dithering=!0),this.alphaTest>0&&(n.alphaTest=this.alphaTest),this.alphaHash===!0&&(n.alphaHash=!0),this.alphaToCoverage===!0&&(n.alphaToCoverage=!0),this.premultipliedAlpha===!0&&(n.premultipliedAlpha=!0),this.forceSinglePass===!0&&(n.forceSinglePass=!0),this.allowOverride===!1&&(n.allowOverride=!1),this.wireframe===!0&&(n.wireframe=!0),this.wireframeLinewidth>1&&(n.wireframeLinewidth=this.wireframeLinewidth),this.wireframeLinecap!=="round"&&(n.wireframeLinecap=this.wireframeLinecap),this.wireframeLinejoin!=="round"&&(n.wireframeLinejoin=this.wireframeLinejoin),this.flatShading===!0&&(n.flatShading=!0),this.visible===!1&&(n.visible=!1),this.toneMapped===!1&&(n.toneMapped=!1),this.fog===!1&&(n.fog=!1),Object.keys(this.userData).length>0&&(n.userData=this.userData);function r(s){const a=[];for(const o in s){const l=s[o];delete l.metadata,a.push(l)}return a}if(t){const s=r(e.textures),a=r(e.images);s.length>0&&(n.textures=s),a.length>0&&(n.images=a)}return n}fromJSON(e,t){if(e.uuid!==void 0&&(this.uuid=e.uuid),e.name!==void 0&&(this.name=e.name),e.color!==void 0&&this.color!==void 0&&this.color.setHex(e.color),e.roughness!==void 0&&(this.roughness=e.roughness),e.metalness!==void 0&&(this.metalness=e.metalness),e.sheen!==void 0&&(this.sheen=e.sheen),e.sheenColor!==void 0&&(this.sheenColor=new ft().setHex(e.sheenColor)),e.sheenRoughness!==void 0&&(this.sheenRoughness=e.sheenRoughness),e.emissive!==void 0&&this.emissive!==void 0&&this.emissive.setHex(e.emissive),e.specular!==void 0&&this.specular!==void 0&&this.specular.setHex(e.specular),e.specularIntensity!==void 0&&(this.specularIntensity=e.specularIntensity),e.specularColor!==void 0&&this.specularColor!==void 0&&this.specularColor.setHex(e.specularColor),e.shininess!==void 0&&(this.shininess=e.shininess),e.clearcoat!==void 0&&(this.clearcoat=e.clearcoat),e.clearcoatRoughness!==void 0&&(this.clearcoatRoughness=e.clearcoatRoughness),e.dispersion!==void 0&&(this.dispersion=e.dispersion),e.iridescence!==void 0&&(this.iridescence=e.iridescence),e.iridescenceIOR!==void 0&&(this.iridescenceIOR=e.iridescenceIOR),e.iridescenceThicknessRange!==void 0&&(this.iridescenceThicknessRange=e.iridescenceThicknessRange),e.transmission!==void 0&&(this.transmission=e.transmission),e.thickness!==void 0&&(this.thickness=e.thickness),e.attenuationDistance!==void 0&&(this.attenuationDistance=e.attenuationDistance),e.attenuationColor!==void 0&&this.attenuationColor!==void 0&&this.attenuationColor.setHex(e.attenuationColor),e.anisotropy!==void 0&&(this.anisotropy=e.anisotropy),e.anisotropyRotation!==void 0&&(this.anisotropyRotation=e.anisotropyRotation),e.fog!==void 0&&(this.fog=e.fog),e.flatShading!==void 0&&(this.flatShading=e.flatShading),e.blending!==void 0&&(this.blending=e.blending),e.combine!==void 0&&(this.combine=e.combine),e.side!==void 0&&(this.side=e.side),e.shadowSide!==void 0&&(this.shadowSide=e.shadowSide),e.opacity!==void 0&&(this.opacity=e.opacity),e.transparent!==void 0&&(this.transparent=e.transparent),e.alphaTest!==void 0&&(this.alphaTest=e.alphaTest),e.alphaHash!==void 0&&(this.alphaHash=e.alphaHash),e.depthFunc!==void 0&&(this.depthFunc=e.depthFunc),e.depthTest!==void 0&&(this.depthTest=e.depthTest),e.depthWrite!==void 0&&(this.depthWrite=e.depthWrite),e.colorWrite!==void 0&&(this.colorWrite=e.colorWrite),e.blendSrc!==void 0&&(this.blendSrc=e.blendSrc),e.blendDst!==void 0&&(this.blendDst=e.blendDst),e.blendEquation!==void 0&&(this.blendEquation=e.blendEquation),e.blendSrcAlpha!==void 0&&(this.blendSrcAlpha=e.blendSrcAlpha),e.blendDstAlpha!==void 0&&(this.blendDstAlpha=e.blendDstAlpha),e.blendEquationAlpha!==void 0&&(this.blendEquationAlpha=e.blendEquationAlpha),e.blendColor!==void 0&&this.blendColor!==void 0&&this.blendColor.setHex(e.blendColor),e.blendAlpha!==void 0&&(this.blendAlpha=e.blendAlpha),e.stencilWriteMask!==void 0&&(this.stencilWriteMask=e.stencilWriteMask),e.stencilFunc!==void 0&&(this.stencilFunc=e.stencilFunc),e.stencilRef!==void 0&&(this.stencilRef=e.stencilRef),e.stencilFuncMask!==void 0&&(this.stencilFuncMask=e.stencilFuncMask),e.stencilFail!==void 0&&(this.stencilFail=e.stencilFail),e.stencilZFail!==void 0&&(this.stencilZFail=e.stencilZFail),e.stencilZPass!==void 0&&(this.stencilZPass=e.stencilZPass),e.stencilWrite!==void 0&&(this.stencilWrite=e.stencilWrite),e.wireframe!==void 0&&(this.wireframe=e.wireframe),e.wireframeLinewidth!==void 0&&(this.wireframeLinewidth=e.wireframeLinewidth),e.wireframeLinecap!==void 0&&(this.wireframeLinecap=e.wireframeLinecap),e.wireframeLinejoin!==void 0&&(this.wireframeLinejoin=e.wireframeLinejoin),e.rotation!==void 0&&(this.rotation=e.rotation),e.linewidth!==void 0&&(this.linewidth=e.linewidth),e.dashSize!==void 0&&(this.dashSize=e.dashSize),e.gapSize!==void 0&&(this.gapSize=e.gapSize),e.scale!==void 0&&(this.scale=e.scale),e.polygonOffset!==void 0&&(this.polygonOffset=e.polygonOffset),e.polygonOffsetFactor!==void 0&&(this.polygonOffsetFactor=e.polygonOffsetFactor),e.polygonOffsetUnits!==void 0&&(this.polygonOffsetUnits=e.polygonOffsetUnits),e.dithering!==void 0&&(this.dithering=e.dithering),e.alphaToCoverage!==void 0&&(this.alphaToCoverage=e.alphaToCoverage),e.premultipliedAlpha!==void 0&&(this.premultipliedAlpha=e.premultipliedAlpha),e.forceSinglePass!==void 0&&(this.forceSinglePass=e.forceSinglePass),e.allowOverride!==void 0&&(this.allowOverride=e.allowOverride),e.visible!==void 0&&(this.visible=e.visible),e.toneMapped!==void 0&&(this.toneMapped=e.toneMapped),e.userData!==void 0&&(this.userData=e.userData),e.vertexColors!==void 0&&(typeof e.vertexColors=="number"?this.vertexColors=e.vertexColors>0:this.vertexColors=e.vertexColors),e.size!==void 0&&(this.size=e.size),e.sizeAttenuation!==void 0&&(this.sizeAttenuation=e.sizeAttenuation),e.map!==void 0&&(this.map=t[e.map]||null),e.matcap!==void 0&&(this.matcap=t[e.matcap]||null),e.alphaMap!==void 0&&(this.alphaMap=t[e.alphaMap]||null),e.bumpMap!==void 0&&(this.bumpMap=t[e.bumpMap]||null),e.bumpScale!==void 0&&(this.bumpScale=e.bumpScale),e.normalMap!==void 0&&(this.normalMap=t[e.normalMap]||null),e.normalMapType!==void 0&&(this.normalMapType=e.normalMapType),e.normalScale!==void 0){let n=e.normalScale;Array.isArray(n)===!1&&(n=[n,n]),this.normalScale=new $e().fromArray(n)}return e.displacementMap!==void 0&&(this.displacementMap=t[e.displacementMap]||null),e.displacementScale!==void 0&&(this.displacementScale=e.displacementScale),e.displacementBias!==void 0&&(this.displacementBias=e.displacementBias),e.roughnessMap!==void 0&&(this.roughnessMap=t[e.roughnessMap]||null),e.metalnessMap!==void 0&&(this.metalnessMap=t[e.metalnessMap]||null),e.emissiveMap!==void 0&&(this.emissiveMap=t[e.emissiveMap]||null),e.emissiveIntensity!==void 0&&(this.emissiveIntensity=e.emissiveIntensity),e.specularMap!==void 0&&(this.specularMap=t[e.specularMap]||null),e.specularIntensityMap!==void 0&&(this.specularIntensityMap=t[e.specularIntensityMap]||null),e.specularColorMap!==void 0&&(this.specularColorMap=t[e.specularColorMap]||null),e.envMap!==void 0&&(this.envMap=t[e.envMap]||null),e.envMapRotation!==void 0&&this.envMapRotation.fromArray(e.envMapRotation),e.envMapIntensity!==void 0&&(this.envMapIntensity=e.envMapIntensity),e.reflectivity!==void 0&&(this.reflectivity=e.reflectivity),e.refractionRatio!==void 0&&(this.refractionRatio=e.refractionRatio),e.lightMap!==void 0&&(this.lightMap=t[e.lightMap]||null),e.lightMapIntensity!==void 0&&(this.lightMapIntensity=e.lightMapIntensity),e.aoMap!==void 0&&(this.aoMap=t[e.aoMap]||null),e.aoMapIntensity!==void 0&&(this.aoMapIntensity=e.aoMapIntensity),e.gradientMap!==void 0&&(this.gradientMap=t[e.gradientMap]||null),e.clearcoatMap!==void 0&&(this.clearcoatMap=t[e.clearcoatMap]||null),e.clearcoatRoughnessMap!==void 0&&(this.clearcoatRoughnessMap=t[e.clearcoatRoughnessMap]||null),e.clearcoatNormalMap!==void 0&&(this.clearcoatNormalMap=t[e.clearcoatNormalMap]||null),e.clearcoatNormalScale!==void 0&&(this.clearcoatNormalScale=new $e().fromArray(e.clearcoatNormalScale)),e.iridescenceMap!==void 0&&(this.iridescenceMap=t[e.iridescenceMap]||null),e.iridescenceThicknessMap!==void 0&&(this.iridescenceThicknessMap=t[e.iridescenceThicknessMap]||null),e.transmissionMap!==void 0&&(this.transmissionMap=t[e.transmissionMap]||null),e.thicknessMap!==void 0&&(this.thicknessMap=t[e.thicknessMap]||null),e.anisotropyMap!==void 0&&(this.anisotropyMap=t[e.anisotropyMap]||null),e.sheenColorMap!==void 0&&(this.sheenColorMap=t[e.sheenColorMap]||null),e.sheenRoughnessMap!==void 0&&(this.sheenRoughnessMap=t[e.sheenRoughnessMap]||null),this}clone(){return new this.constructor().copy(this)}copy(e){this.name=e.name,this.blending=e.blending,this.side=e.side,this.vertexColors=e.vertexColors,this.opacity=e.opacity,this.transparent=e.transparent,this.blendSrc=e.blendSrc,this.blendDst=e.blendDst,this.blendEquation=e.blendEquation,this.blendSrcAlpha=e.blendSrcAlpha,this.blendDstAlpha=e.blendDstAlpha,this.blendEquationAlpha=e.blendEquationAlpha,this.blendColor.copy(e.blendColor),this.blendAlpha=e.blendAlpha,this.depthFunc=e.depthFunc,this.depthTest=e.depthTest,this.depthWrite=e.depthWrite,this.stencilWriteMask=e.stencilWriteMask,this.stencilFunc=e.stencilFunc,this.stencilRef=e.stencilRef,this.stencilFuncMask=e.stencilFuncMask,this.stencilFail=e.stencilFail,this.stencilZFail=e.stencilZFail,this.stencilZPass=e.stencilZPass,this.stencilWrite=e.stencilWrite;const t=e.clippingPlanes;let n=null;if(t!==null){const r=t.length;n=new Array(r);for(let s=0;s!==r;++s)n[s]=t[s].clone()}return this.clippingPlanes=n,this.clipIntersection=e.clipIntersection,this.clipShadows=e.clipShadows,this.shadowSide=e.shadowSide,this.colorWrite=e.colorWrite,this.precision=e.precision,this.polygonOffset=e.polygonOffset,this.polygonOffsetFactor=e.polygonOffsetFactor,this.polygonOffsetUnits=e.polygonOffsetUnits,this.dithering=e.dithering,this.alphaTest=e.alphaTest,this.alphaHash=e.alphaHash,this.alphaToCoverage=e.alphaToCoverage,this.premultipliedAlpha=e.premultipliedAlpha,this.forceSinglePass=e.forceSinglePass,this.allowOverride=e.allowOverride,this.visible=e.visible,this.toneMapped=e.toneMapped,this.userData=JSON.parse(JSON.stringify(e.userData)),this}dispose(){this.dispatchEvent({type:"dispose"})}set needsUpdate(e){e===!0&&this.version++}}class jd extends Bs{constructor(e){super(),this.isSpriteMaterial=!0,this.type="SpriteMaterial",this.color=new ft(16777215),this.map=null,this.alphaMap=null,this.rotation=0,this.sizeAttenuation=!0,this.transparent=!0,this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.alphaMap=e.alphaMap,this.rotation=e.rotation,this.sizeAttenuation=e.sizeAttenuation,this.fog=e.fog,this}}let ds;const na=new X,ps=new X,ms=new X,gs=new $e,ia=new $e,Qd=new Xt,za=new X,ra=new X,ka=new X,jf=new $e,Al=new $e,Qf=new $e;class q0 extends _n{constructor(e=new jd){if(super(),this.isSprite=!0,this.type="Sprite",ds===void 0){ds=new gn;const t=new Float32Array([-.5,-.5,0,0,0,.5,-.5,0,1,0,.5,.5,0,1,1,-.5,.5,0,0,1]),n=new X0(t,5);ds.setIndex([0,1,2,0,2,3]),ds.setAttribute("position",new To(n,3,0,!1)),ds.setAttribute("uv",new To(n,2,3,!1))}this.geometry=ds,this.material=e,this.center=new $e(.5,.5),this.count=1}raycast(e,t){e.camera===null&&gt('Sprite: "Raycaster.camera" needs to be set in order to raycast against sprites.'),ps.setFromMatrixScale(this.matrixWorld),Qd.copy(e.camera.matrixWorld),this.modelViewMatrix.multiplyMatrices(e.camera.matrixWorldInverse,this.matrixWorld),ms.setFromMatrixPosition(this.modelViewMatrix),e.camera.isPerspectiveCamera&&this.material.sizeAttenuation===!1&&ps.multiplyScalar(-ms.z);const n=this.material.rotation;let r,s;n!==0&&(s=Math.cos(n),r=Math.sin(n));const a=this.center;Va(za.set(-.5,-.5,0),ms,a,ps,r,s),Va(ra.set(.5,-.5,0),ms,a,ps,r,s),Va(ka.set(.5,.5,0),ms,a,ps,r,s),jf.set(0,0),Al.set(1,0),Qf.set(1,1);let o=e.ray.intersectTriangle(za,ra,ka,!1,na);if(o===null&&(Va(ra.set(-.5,.5,0),ms,a,ps,r,s),Al.set(0,1),o=e.ray.intersectTriangle(za,ka,ra,!1,na),o===null))return;const l=e.ray.origin.distanceTo(na);l<e.near||l>e.far||t.push({distance:l,point:na.clone(),uv:li.getInterpolation(na,za,ra,ka,jf,Al,Qf,new $e),face:null,object:this})}copy(e,t){return super.copy(e,t),e.center!==void 0&&this.center.copy(e.center),this.material=e.material,this}}function Va(i,e,t,n,r,s){gs.subVectors(i,t).addScalar(.5).multiply(n),r!==void 0?(ia.x=s*gs.x-r*gs.y,ia.y=r*gs.x+s*gs.y):ia.copy(gs),i.copy(e),i.x+=ia.x,i.y+=ia.y,i.applyMatrix4(Qd)}const Wi=new X,wl=new X,Ga=new X,cr=new X,Rl=new X,Ha=new X,Cl=new X;class pu{constructor(e=new X,t=new X(0,0,-1)){this.origin=e,this.direction=t}set(e,t){return this.origin.copy(e),this.direction.copy(t),this}copy(e){return this.origin.copy(e.origin),this.direction.copy(e.direction),this}at(e,t){return t.copy(this.origin).addScaledVector(this.direction,e)}lookAt(e){return this.direction.copy(e).sub(this.origin).normalize(),this}recast(e){return this.origin.copy(this.at(e,Wi)),this}closestPointToPoint(e,t){t.subVectors(e,this.origin);const n=t.dot(this.direction);return n<0?t.copy(this.origin):t.copy(this.origin).addScaledVector(this.direction,n)}distanceToPoint(e){return Math.sqrt(this.distanceSqToPoint(e))}distanceSqToPoint(e){const t=Wi.subVectors(e,this.origin).dot(this.direction);return t<0?this.origin.distanceToSquared(e):(Wi.copy(this.origin).addScaledVector(this.direction,t),Wi.distanceToSquared(e))}distanceSqToSegment(e,t,n,r){wl.copy(e).add(t).multiplyScalar(.5),Ga.copy(t).sub(e).normalize(),cr.copy(this.origin).sub(wl);const s=e.distanceTo(t)*.5,a=-this.direction.dot(Ga),o=cr.dot(this.direction),l=-cr.dot(Ga),c=cr.lengthSq(),f=Math.abs(1-a*a);let h,u,d,x;if(f>0)if(h=a*l-o,u=a*o-l,x=s*f,h>=0)if(u>=-x)if(u<=x){const E=1/f;h*=E,u*=E,d=h*(h+a*u+2*o)+u*(a*h+u+2*l)+c}else u=s,h=Math.max(0,-(a*u+o)),d=-h*h+u*(u+2*l)+c;else u=-s,h=Math.max(0,-(a*u+o)),d=-h*h+u*(u+2*l)+c;else u<=-x?(h=Math.max(0,-(-a*s+o)),u=h>0?-s:Math.min(Math.max(-s,-l),s),d=-h*h+u*(u+2*l)+c):u<=x?(h=0,u=Math.min(Math.max(-s,-l),s),d=u*(u+2*l)+c):(h=Math.max(0,-(a*s+o)),u=h>0?s:Math.min(Math.max(-s,-l),s),d=-h*h+u*(u+2*l)+c);else u=a>0?-s:s,h=Math.max(0,-(a*u+o)),d=-h*h+u*(u+2*l)+c;return n&&n.copy(this.origin).addScaledVector(this.direction,h),r&&r.copy(wl).addScaledVector(Ga,u),d}intersectSphere(e,t){Wi.subVectors(e.center,this.origin);const n=Wi.dot(this.direction),r=Wi.dot(Wi)-n*n,s=e.radius*e.radius;if(r>s)return null;const a=Math.sqrt(s-r),o=n-a,l=n+a;return l<0?null:o<0?this.at(l,t):this.at(o,t)}intersectsSphere(e){return e.radius<0?!1:this.distanceSqToPoint(e.center)<=e.radius*e.radius}distanceToPlane(e){const t=e.normal.dot(this.direction);if(t===0)return e.distanceToPoint(this.origin)===0?0:null;const n=-(this.origin.dot(e.normal)+e.constant)/t;return n>=0?n:null}intersectPlane(e,t){const n=this.distanceToPlane(e);return n===null?null:this.at(n,t)}intersectsPlane(e){const t=e.distanceToPoint(this.origin);return t===0||e.normal.dot(this.direction)*t<0}intersectBox(e,t){let n,r,s,a,o,l;const c=1/this.direction.x,f=1/this.direction.y,h=1/this.direction.z,u=this.origin;return c>=0?(n=(e.min.x-u.x)*c,r=(e.max.x-u.x)*c):(n=(e.max.x-u.x)*c,r=(e.min.x-u.x)*c),f>=0?(s=(e.min.y-u.y)*f,a=(e.max.y-u.y)*f):(s=(e.max.y-u.y)*f,a=(e.min.y-u.y)*f),n>a||s>r||((s>n||isNaN(n))&&(n=s),(a<r||isNaN(r))&&(r=a),h>=0?(o=(e.min.z-u.z)*h,l=(e.max.z-u.z)*h):(o=(e.max.z-u.z)*h,l=(e.min.z-u.z)*h),n>l||o>r)||((o>n||n!==n)&&(n=o),(l<r||r!==r)&&(r=l),r<0)?null:this.at(n>=0?n:r,t)}intersectsBox(e){return this.intersectBox(e,Wi)!==null}intersectTriangle(e,t,n,r,s){Rl.subVectors(t,e),Ha.subVectors(n,e),Cl.crossVectors(Rl,Ha);let a=this.direction.dot(Cl),o;if(a>0){if(r)return null;o=1}else if(a<0)o=-1,a=-a;else return null;cr.subVectors(this.origin,e);const l=o*this.direction.dot(Ha.crossVectors(cr,Ha));if(l<0)return null;const c=o*this.direction.dot(Rl.cross(cr));if(c<0||l+c>a)return null;const f=-o*cr.dot(Cl);return f<0?null:this.at(f/a,s)}applyMatrix4(e){return this.origin.applyMatrix4(e),this.direction.transformDirection(e),this}equals(e){return e.origin.equals(this.origin)&&e.direction.equals(this.direction)}clone(){return new this.constructor().copy(this)}}class mu extends Bs{constructor(e){super(),this.isMeshBasicMaterial=!0,this.type="MeshBasicMaterial",this.color=new ft(16777215),this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.specularMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new Gr,this.combine=Pd,this.reflectivity=1,this.refractionRatio=.98,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.lightMap=e.lightMap,this.lightMapIntensity=e.lightMapIntensity,this.aoMap=e.aoMap,this.aoMapIntensity=e.aoMapIntensity,this.specularMap=e.specularMap,this.alphaMap=e.alphaMap,this.envMap=e.envMap,this.envMapRotation.copy(e.envMapRotation),this.combine=e.combine,this.reflectivity=e.reflectivity,this.refractionRatio=e.refractionRatio,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.wireframeLinecap=e.wireframeLinecap,this.wireframeLinejoin=e.wireframeLinejoin,this.fog=e.fog,this}}const eh=new Xt,Rr=new pu,Wa=new Bo,th=new X,Xa=new X,Ya=new X,qa=new X,Pl=new X,Ka=new X,nh=new X,$a=new X;class vi extends _n{constructor(e=new gn,t=new mu){super(),this.isMesh=!0,this.type="Mesh",this.geometry=e,this.material=t,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.count=1,this.updateMorphTargets()}copy(e,t){return super.copy(e,t),e.morphTargetInfluences!==void 0&&(this.morphTargetInfluences=e.morphTargetInfluences.slice()),e.morphTargetDictionary!==void 0&&(this.morphTargetDictionary=Object.assign({},e.morphTargetDictionary)),this.material=Array.isArray(e.material)?e.material.slice():e.material,this.geometry=e.geometry,this}updateMorphTargets(){const t=this.geometry.morphAttributes,n=Object.keys(t);if(n.length>0){const r=t[n[0]];if(r!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let s=0,a=r.length;s<a;s++){const o=r[s].name||String(s);this.morphTargetInfluences.push(0),this.morphTargetDictionary[o]=s}}}}getVertexPosition(e,t){const n=this.geometry,r=n.attributes.position,s=n.morphAttributes.position,a=n.morphTargetsRelative;t.fromBufferAttribute(r,e);const o=this.morphTargetInfluences;if(s&&o){Ka.set(0,0,0);for(let l=0,c=s.length;l<c;l++){const f=o[l],h=s[l];f!==0&&(Pl.fromBufferAttribute(h,e),a?Ka.addScaledVector(Pl,f):Ka.addScaledVector(Pl.sub(t),f))}t.add(Ka)}return t}raycast(e,t){const n=this.geometry,r=this.material,s=this.matrixWorld;r!==void 0&&(n.boundingSphere===null&&n.computeBoundingSphere(),Wa.copy(n.boundingSphere),Wa.applyMatrix4(s),Rr.copy(e.ray).recast(e.near),!(Wa.containsPoint(Rr.origin)===!1&&(Rr.intersectSphere(Wa,th)===null||Rr.origin.distanceToSquared(th)>(e.far-e.near)**2))&&(eh.copy(s).invert(),Rr.copy(e.ray).applyMatrix4(eh),!(n.boundingBox!==null&&Rr.intersectsBox(n.boundingBox)===!1)&&this._computeIntersections(e,t,Rr)))}_computeIntersections(e,t,n){let r;const s=this.geometry,a=this.material,o=s.index,l=s.attributes.position,c=s.attributes.uv,f=s.attributes.uv1,h=s.attributes.normal,u=s.groups,d=s.drawRange;if(o!==null)if(Array.isArray(a))for(let x=0,E=u.length;x<E;x++){const m=u[x],p=a[m.materialIndex],T=Math.max(m.start,d.start),P=Math.min(o.count,Math.min(m.start+m.count,d.start+d.count));for(let S=T,C=P;S<C;S+=3){const y=o.getX(S),I=o.getX(S+1),v=o.getX(S+2);r=Za(this,p,e,n,c,f,h,y,I,v),r&&(r.faceIndex=Math.floor(S/3),r.face.materialIndex=m.materialIndex,t.push(r))}}else{const x=Math.max(0,d.start),E=Math.min(o.count,d.start+d.count);for(let m=x,p=E;m<p;m+=3){const T=o.getX(m),P=o.getX(m+1),S=o.getX(m+2);r=Za(this,a,e,n,c,f,h,T,P,S),r&&(r.faceIndex=Math.floor(m/3),t.push(r))}}else if(l!==void 0)if(Array.isArray(a))for(let x=0,E=u.length;x<E;x++){const m=u[x],p=a[m.materialIndex],T=Math.max(m.start,d.start),P=Math.min(l.count,Math.min(m.start+m.count,d.start+d.count));for(let S=T,C=P;S<C;S+=3){const y=S,I=S+1,v=S+2;r=Za(this,p,e,n,c,f,h,y,I,v),r&&(r.faceIndex=Math.floor(S/3),r.face.materialIndex=m.materialIndex,t.push(r))}}else{const x=Math.max(0,d.start),E=Math.min(l.count,d.start+d.count);for(let m=x,p=E;m<p;m+=3){const T=m,P=m+1,S=m+2;r=Za(this,a,e,n,c,f,h,T,P,S),r&&(r.faceIndex=Math.floor(m/3),t.push(r))}}}}function K0(i,e,t,n,r,s,a,o){let l;if(e.side===Vn?l=n.intersectTriangle(a,s,r,!0,o):l=n.intersectTriangle(r,s,a,e.side===xr,o),l===null)return null;$a.copy(o),$a.applyMatrix4(i.matrixWorld);const c=t.ray.origin.distanceTo($a);return c<t.near||c>t.far?null:{distance:c,point:$a.clone(),object:i}}function Za(i,e,t,n,r,s,a,o,l,c){i.getVertexPosition(o,Xa),i.getVertexPosition(l,Ya),i.getVertexPosition(c,qa);const f=K0(i,e,t,n,Xa,Ya,qa,nh);if(f){const h=new X;li.getBarycoord(nh,Xa,Ya,qa,h),r&&(f.uv=li.getInterpolatedAttribute(r,o,l,c,h,new $e)),s&&(f.uv1=li.getInterpolatedAttribute(s,o,l,c,h,new $e)),a&&(f.normal=li.getInterpolatedAttribute(a,o,l,c,h,new X),f.normal.dot(n.direction)>0&&f.normal.multiplyScalar(-1));const u={a:o,b:l,c,normal:new X,materialIndex:0};li.getNormal(Xa,Ya,qa,u.normal),f.face=u,f.barycoord=h}return f}class $0 extends wn{constructor(e=null,t=1,n=1,r,s,a,o,l,c=Mn,f=Mn,h,u){super(null,a,o,l,c,f,r,s,h,u),this.isDataTexture=!0,this.image={data:e,width:t,height:n},this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}}const Dl=new X,Z0=new X,J0=new nt;class hr{constructor(e=new X(1,0,0),t=0){this.isPlane=!0,this.normal=e,this.constant=t}set(e,t){return this.normal.copy(e),this.constant=t,this}setComponents(e,t,n,r){return this.normal.set(e,t,n),this.constant=r,this}setFromNormalAndCoplanarPoint(e,t){return this.normal.copy(e),this.constant=-t.dot(this.normal),this}setFromCoplanarPoints(e,t,n){const r=Dl.subVectors(n,t).cross(Z0.subVectors(e,t)).normalize();return this.setFromNormalAndCoplanarPoint(r,e),this}copy(e){return this.normal.copy(e.normal),this.constant=e.constant,this}normalize(){const e=1/this.normal.length();return this.normal.multiplyScalar(e),this.constant*=e,this}negate(){return this.constant*=-1,this.normal.negate(),this}distanceToPoint(e){return this.normal.dot(e)+this.constant}distanceToSphere(e){return this.distanceToPoint(e.center)-e.radius}projectPoint(e,t){return t.copy(e).addScaledVector(this.normal,-this.distanceToPoint(e))}intersectLine(e,t,n=!0){const r=e.delta(Dl),s=this.normal.dot(r);if(s===0)return this.distanceToPoint(e.start)===0?t.copy(e.start):null;const a=-(e.start.dot(this.normal)+this.constant)/s;return n===!0&&(a<0||a>1)?null:t.copy(e.start).addScaledVector(r,a)}intersectsLine(e){const t=this.distanceToPoint(e.start),n=this.distanceToPoint(e.end);return t<0&&n>0||n<0&&t>0}intersectsBox(e){return e.intersectsPlane(this)}intersectsSphere(e){return e.intersectsPlane(this)}coplanarPoint(e){return e.copy(this.normal).multiplyScalar(-this.constant)}applyMatrix4(e,t){const n=t||J0.getNormalMatrix(e),r=this.coplanarPoint(Dl).applyMatrix4(e),s=this.normal.applyMatrix3(n).normalize();return this.constant=-r.dot(s),this}translate(e){return this.constant-=e.dot(this.normal),this}equals(e){return e.normal.equals(this.normal)&&e.constant===this.constant}clone(){return new this.constructor().copy(this)}}const Cr=new Bo,j0=new $e(.5,.5),Ja=new X;class gu{constructor(e=new hr,t=new hr,n=new hr,r=new hr,s=new hr,a=new hr){this.planes=[e,t,n,r,s,a]}set(e,t,n,r,s,a){const o=this.planes;return o[0].copy(e),o[1].copy(t),o[2].copy(n),o[3].copy(r),o[4].copy(s),o[5].copy(a),this}copy(e){const t=this.planes;for(let n=0;n<6;n++)t[n].copy(e.planes[n]);return this}setFromProjectionMatrix(e,t=Ii,n=!1){const r=this.planes,s=e.elements,a=s[0],o=s[1],l=s[2],c=s[3],f=s[4],h=s[5],u=s[6],d=s[7],x=s[8],E=s[9],m=s[10],p=s[11],T=s[12],P=s[13],S=s[14],C=s[15];if(r[0].setComponents(c-a,d-f,p-x,C-T).normalize(),r[1].setComponents(c+a,d+f,p+x,C+T).normalize(),r[2].setComponents(c+o,d+h,p+E,C+P).normalize(),r[3].setComponents(c-o,d-h,p-E,C-P).normalize(),n)r[4].setComponents(l,u,m,S).normalize(),r[5].setComponents(c-l,d-u,p-m,C-S).normalize();else if(r[4].setComponents(c-l,d-u,p-m,C-S).normalize(),t===Ii)r[5].setComponents(c+l,d+u,p+m,C+S).normalize();else if(t===va)r[5].setComponents(l,u,m,S).normalize();else throw new Error("THREE.Frustum.setFromProjectionMatrix(): Invalid coordinate system: "+t);return this}intersectsObject(e){if(e.boundingSphere!==void 0)e.boundingSphere===null&&e.computeBoundingSphere(),Cr.copy(e.boundingSphere).applyMatrix4(e.matrixWorld);else{const t=e.geometry;t.boundingSphere===null&&t.computeBoundingSphere(),Cr.copy(t.boundingSphere).applyMatrix4(e.matrixWorld)}return this.intersectsSphere(Cr)}intersectsSprite(e){Cr.center.set(0,0,0);const t=j0.distanceTo(e.center);return Cr.radius=.7071067811865476+t,Cr.applyMatrix4(e.matrixWorld),this.intersectsSphere(Cr)}intersectsSphere(e){const t=this.planes,n=e.center,r=-e.radius;for(let s=0;s<6;s++)if(t[s].distanceToPoint(n)<r)return!1;return!0}intersectsBox(e){const t=this.planes;for(let n=0;n<6;n++){const r=t[n];if(Ja.x=r.normal.x>0?e.max.x:e.min.x,Ja.y=r.normal.y>0?e.max.y:e.min.y,Ja.z=r.normal.z>0?e.max.z:e.min.z,r.distanceToPoint(Ja)<0)return!1}return!0}containsPoint(e){const t=this.planes;for(let n=0;n<6;n++)if(t[n].distanceToPoint(e)<0)return!1;return!0}clone(){return new this.constructor().copy(this)}}class zo extends Bs{constructor(e){super(),this.isLineBasicMaterial=!0,this.type="LineBasicMaterial",this.color=new ft(16777215),this.map=null,this.linewidth=1,this.linecap="round",this.linejoin="round",this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.linewidth=e.linewidth,this.linecap=e.linecap,this.linejoin=e.linejoin,this.fog=e.fog,this}}const Ao=new X,wo=new X,ih=new Xt,sa=new pu,ja=new Bo,Ll=new X,rh=new X;class po extends _n{constructor(e=new gn,t=new zo){super(),this.isLine=!0,this.type="Line",this.geometry=e,this.material=t,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.updateMorphTargets()}copy(e,t){return super.copy(e,t),this.material=Array.isArray(e.material)?e.material.slice():e.material,this.geometry=e.geometry,this}computeLineDistances(){const e=this.geometry;if(e.index===null){const t=e.attributes.position,n=[0];for(let r=1,s=t.count;r<s;r++)Ao.fromBufferAttribute(t,r-1),wo.fromBufferAttribute(t,r),n[r]=n[r-1],n[r]+=Ao.distanceTo(wo);e.setAttribute("lineDistance",new Bn(n,1))}else Je("Line.computeLineDistances(): Computation only possible with non-indexed BufferGeometry.");return this}raycast(e,t){const n=this.geometry,r=this.matrixWorld,s=e.params.Line.threshold,a=n.drawRange;if(n.boundingSphere===null&&n.computeBoundingSphere(),ja.copy(n.boundingSphere),ja.applyMatrix4(r),ja.radius+=s,e.ray.intersectsSphere(ja)===!1)return;ih.copy(r).invert(),sa.copy(e.ray).applyMatrix4(ih);const o=s/((this.scale.x+this.scale.y+this.scale.z)/3),l=o*o,c=this.isLineSegments?2:1,f=n.index,u=n.attributes.position;if(f!==null){const d=Math.max(0,a.start),x=Math.min(f.count,a.start+a.count);for(let E=d,m=x-1;E<m;E+=c){const p=f.getX(E),T=f.getX(E+1),P=Qa(this,e,sa,l,p,T,E);P&&t.push(P)}if(this.isLineLoop){const E=f.getX(x-1),m=f.getX(d),p=Qa(this,e,sa,l,E,m,x-1);p&&t.push(p)}}else{const d=Math.max(0,a.start),x=Math.min(u.count,a.start+a.count);for(let E=d,m=x-1;E<m;E+=c){const p=Qa(this,e,sa,l,E,E+1,E);p&&t.push(p)}if(this.isLineLoop){const E=Qa(this,e,sa,l,x-1,d,x-1);E&&t.push(E)}}}updateMorphTargets(){const t=this.geometry.morphAttributes,n=Object.keys(t);if(n.length>0){const r=t[n[0]];if(r!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let s=0,a=r.length;s<a;s++){const o=r[s].name||String(s);this.morphTargetInfluences.push(0),this.morphTargetDictionary[o]=s}}}}}function Qa(i,e,t,n,r,s,a){const o=i.geometry.attributes.position;if(Ao.fromBufferAttribute(o,r),wo.fromBufferAttribute(o,s),t.distanceSqToSegment(Ao,wo,Ll,rh)>n)return;Ll.applyMatrix4(i.matrixWorld);const c=e.ray.origin.distanceTo(Ll);if(!(c<e.near||c>e.far))return{distance:c,point:rh.clone().applyMatrix4(i.matrixWorld),index:a,face:null,faceIndex:null,barycoord:null,object:i}}const sh=new X,ah=new X;class ep extends po{constructor(e,t){super(e,t),this.isLineSegments=!0,this.type="LineSegments"}computeLineDistances(){const e=this.geometry;if(e.index===null){const t=e.attributes.position,n=[];for(let r=0,s=t.count;r<s;r+=2)sh.fromBufferAttribute(t,r),ah.fromBufferAttribute(t,r+1),n[r]=r===0?0:n[r-1],n[r+1]=n[r]+sh.distanceTo(ah);e.setAttribute("lineDistance",new Bn(n,1))}else Je("LineSegments.computeLineDistances(): Computation only possible with non-indexed BufferGeometry.");return this}}class tp extends wn{constructor(e=[],t=kr,n,r,s,a,o,l,c,f){super(e,t,n,r,s,a,o,l,c,f),this.isCubeTexture=!0,this.flipY=!1}get images(){return this.image}set images(e){this.image=e}}class Q0 extends wn{constructor(e,t,n,r,s,a,o,l,c){super(e,t,n,r,s,a,o,l,c),this.isCanvasTexture=!0,this.needsUpdate=!0}}class Is extends wn{constructor(e,t,n=Fi,r,s,a,o=Mn,l=Mn,c,f=Ji,h=1){if(f!==Ji&&f!==Or)throw new Error("THREE.DepthTexture: format must be either THREE.DepthFormat or THREE.DepthStencilFormat");const u={width:e,height:t,depth:h};super(u,r,s,a,o,l,f,n,c),this.isDepthTexture=!0,this.flipY=!1,this.generateMipmaps=!1,this.compareFunction=null}copy(e){return super.copy(e),this.source=new du(Object.assign({},e.image)),this.compareFunction=e.compareFunction,this}toJSON(e){const t=super.toJSON(e);return this.compareFunction!==null&&(t.compareFunction=this.compareFunction),t}}class e_ extends Is{constructor(e,t=Fi,n=kr,r,s,a=Mn,o=Mn,l,c=Ji){const f={width:e,height:e,depth:1},h=[f,f,f,f,f,f];super(e,e,t,n,r,s,a,o,l,c),this.image=h,this.isCubeDepthTexture=!0,this.isCubeTexture=!0}get images(){return this.image}set images(e){this.image=e}}class np extends wn{constructor(e=null){super(),this.sourceTexture=e,this.isExternalTexture=!0}copy(e){return super.copy(e),this.sourceTexture=e.sourceTexture,this}}class Ma extends gn{constructor(e=1,t=1,n=1,r=1,s=1,a=1){super(),this.type="BoxGeometry",this.parameters={width:e,height:t,depth:n,widthSegments:r,heightSegments:s,depthSegments:a};const o=this;r=Math.floor(r),s=Math.floor(s),a=Math.floor(a);const l=[],c=[],f=[],h=[];let u=0,d=0;x("z","y","x",-1,-1,n,t,e,a,s,0),x("z","y","x",1,-1,n,t,-e,a,s,1),x("x","z","y",1,1,e,n,t,r,a,2),x("x","z","y",1,-1,e,n,-t,r,a,3),x("x","y","z",1,-1,e,t,n,r,s,4),x("x","y","z",-1,-1,e,t,-n,r,s,5),this.setIndex(l),this.setAttribute("position",new Bn(c,3)),this.setAttribute("normal",new Bn(f,3)),this.setAttribute("uv",new Bn(h,2));function x(E,m,p,T,P,S,C,y,I,v,A){const F=S/I,D=C/v,z=S/2,O=C/2,k=y/2,L=I+1,N=v+1;let B=0,j=0;const ne=new X;for(let ae=0;ae<N;ae++){const fe=ae*D-O;for(let re=0;re<L;re++){const le=re*F-z;ne[E]=le*T,ne[m]=fe*P,ne[p]=k,c.push(ne.x,ne.y,ne.z),ne[E]=0,ne[m]=0,ne[p]=y>0?1:-1,f.push(ne.x,ne.y,ne.z),h.push(re/I),h.push(1-ae/v),B+=1}}for(let ae=0;ae<v;ae++)for(let fe=0;fe<I;fe++){const re=u+fe+L*ae,le=u+fe+L*(ae+1),Ze=u+(fe+1)+L*(ae+1),Ke=u+(fe+1)+L*ae;l.push(re,le,Ke),l.push(le,Ze,Ke),j+=6}o.addGroup(d,j,A),d+=j,u+=B}}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new Ma(e.width,e.height,e.depth,e.widthSegments,e.heightSegments,e.depthSegments)}}class ko extends gn{constructor(e=1,t=1,n=1,r=1){super(),this.type="PlaneGeometry",this.parameters={width:e,height:t,widthSegments:n,heightSegments:r};const s=e/2,a=t/2,o=Math.floor(n),l=Math.floor(r),c=o+1,f=l+1,h=e/o,u=t/l,d=[],x=[],E=[],m=[];for(let p=0;p<f;p++){const T=p*u-a;for(let P=0;P<c;P++){const S=P*h-s;x.push(S,-T,0),E.push(0,0,1),m.push(P/o),m.push(1-p/l)}}for(let p=0;p<l;p++)for(let T=0;T<o;T++){const P=T+c*p,S=T+c*(p+1),C=T+1+c*(p+1),y=T+1+c*p;d.push(P,S,y),d.push(S,C,y)}this.setIndex(d),this.setAttribute("position",new Bn(x,3)),this.setAttribute("normal",new Bn(E,3)),this.setAttribute("uv",new Bn(m,2))}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new ko(e.width,e.height,e.widthSegments,e.heightSegments)}}function Us(i){const e={};for(const t in i){e[t]={};for(const n in i[t]){const r=i[t][n];if(oh(r))r.isRenderTargetTexture?(Je("UniformsUtils: Textures of render targets cannot be cloned via cloneUniforms() or mergeUniforms()."),e[t][n]=null):e[t][n]=r.clone();else if(Array.isArray(r))if(oh(r[0])){const s=[];for(let a=0,o=r.length;a<o;a++)s[a]=r[a].clone();e[t][n]=s}else e[t][n]=r.slice();else e[t][n]=r}}return e}function Fn(i){const e={};for(let t=0;t<i.length;t++){const n=Us(i[t]);for(const r in n)e[r]=n[r]}return e}function oh(i){return i&&(i.isColor||i.isMatrix3||i.isMatrix4||i.isVector2||i.isVector3||i.isVector4||i.isTexture||i.isQuaternion)}function t_(i){const e=[];for(let t=0;t<i.length;t++)e.push(i[t].clone());return e}function ip(i){const e=i.getRenderTarget();return e===null?i.outputColorSpace:e.isXRRenderTarget===!0?e.texture.colorSpace:mt.workingColorSpace}const n_={clone:Us,merge:Fn};var i_=`void main() {
	gl_Position = projectionMatrix * modelViewMatrix * vec4( position, 1.0 );
}`,r_=`void main() {
	gl_FragColor = vec4( 1.0, 0.0, 0.0, 1.0 );
}`;class Oi extends Bs{constructor(e){super(),this.isShaderMaterial=!0,this.type="ShaderMaterial",this.defines={},this.uniforms={},this.uniformsGroups=[],this.vertexShader=i_,this.fragmentShader=r_,this.linewidth=1,this.wireframe=!1,this.wireframeLinewidth=1,this.fog=!1,this.lights=!1,this.clipping=!1,this.forceSinglePass=!0,this.extensions={clipCullDistance:!1,multiDraw:!1},this.defaultAttributeValues={color:[1,1,1],uv:[0,0],uv1:[0,0]},this.index0AttributeName=void 0,this.uniformsNeedUpdate=!1,this.glslVersion=null,e!==void 0&&this.setValues(e)}copy(e){return super.copy(e),this.fragmentShader=e.fragmentShader,this.vertexShader=e.vertexShader,this.uniforms=Us(e.uniforms),this.uniformsGroups=t_(e.uniformsGroups),this.defines=Object.assign({},e.defines),this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.fog=e.fog,this.lights=e.lights,this.clipping=e.clipping,this.extensions=Object.assign({},e.extensions),this.glslVersion=e.glslVersion,this.defaultAttributeValues=Object.assign({},e.defaultAttributeValues),this.index0AttributeName=e.index0AttributeName,this.uniformsNeedUpdate=e.uniformsNeedUpdate,this}toJSON(e){const t=super.toJSON(e);t.glslVersion=this.glslVersion,t.uniforms={};for(const r in this.uniforms){const a=this.uniforms[r].value;a&&a.isTexture?t.uniforms[r]={type:"t",value:a.toJSON(e).uuid}:a&&a.isColor?t.uniforms[r]={type:"c",value:a.getHex()}:a&&a.isVector2?t.uniforms[r]={type:"v2",value:a.toArray()}:a&&a.isVector3?t.uniforms[r]={type:"v3",value:a.toArray()}:a&&a.isVector4?t.uniforms[r]={type:"v4",value:a.toArray()}:a&&a.isMatrix3?t.uniforms[r]={type:"m3",value:a.toArray()}:a&&a.isMatrix4?t.uniforms[r]={type:"m4",value:a.toArray()}:t.uniforms[r]={value:a}}Object.keys(this.defines).length>0&&(t.defines=this.defines),t.vertexShader=this.vertexShader,t.fragmentShader=this.fragmentShader,t.lights=this.lights,t.clipping=this.clipping;const n={};for(const r in this.extensions)this.extensions[r]===!0&&(n[r]=!0);return Object.keys(n).length>0&&(t.extensions=n),t}fromJSON(e,t){if(super.fromJSON(e,t),e.uniforms!==void 0)for(const n in e.uniforms){const r=e.uniforms[n];switch(this.uniforms[n]={},r.type){case"t":this.uniforms[n].value=t[r.value]||null;break;case"c":this.uniforms[n].value=new ft().setHex(r.value);break;case"v2":this.uniforms[n].value=new $e().fromArray(r.value);break;case"v3":this.uniforms[n].value=new X().fromArray(r.value);break;case"v4":this.uniforms[n].value=new $t().fromArray(r.value);break;case"m3":this.uniforms[n].value=new nt().fromArray(r.value);break;case"m4":this.uniforms[n].value=new Xt().fromArray(r.value);break;default:this.uniforms[n].value=r.value}}if(e.defines!==void 0&&(this.defines=e.defines),e.vertexShader!==void 0&&(this.vertexShader=e.vertexShader),e.fragmentShader!==void 0&&(this.fragmentShader=e.fragmentShader),e.glslVersion!==void 0&&(this.glslVersion=e.glslVersion),e.extensions!==void 0)for(const n in e.extensions)this.extensions[n]=e.extensions[n];return e.lights!==void 0&&(this.lights=e.lights),e.clipping!==void 0&&(this.clipping=e.clipping),this}}class s_ extends Oi{constructor(e){super(e),this.isRawShaderMaterial=!0,this.type="RawShaderMaterial"}}class a_ extends Bs{constructor(e){super(),this.isMeshDepthMaterial=!0,this.type="MeshDepthMaterial",this.depthPacking=p0,this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.wireframe=!1,this.wireframeLinewidth=1,this.setValues(e)}copy(e){return super.copy(e),this.depthPacking=e.depthPacking,this.map=e.map,this.alphaMap=e.alphaMap,this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this}}class o_ extends Bs{constructor(e){super(),this.isMeshDistanceMaterial=!0,this.type="MeshDistanceMaterial",this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.setValues(e)}copy(e){return super.copy(e),this.map=e.map,this.alphaMap=e.alphaMap,this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this}}class rp extends _n{constructor(e,t=1){super(),this.isLight=!0,this.type="Light",this.color=new ft(e),this.intensity=t}dispose(){this.dispatchEvent({type:"dispose"})}copy(e,t){return super.copy(e,t),this.color.copy(e.color),this.intensity=e.intensity,this}toJSON(e){const t=super.toJSON(e);return t.object.color=this.color.getHex(),t.object.intensity=this.intensity,t}}const Il=new Xt,lh=new X,ch=new X;class l_{constructor(e){this.camera=e,this.intensity=1,this.bias=0,this.biasNode=null,this.normalBias=0,this.radius=1,this.blurSamples=8,this.mapSize=new $e(512,512),this.mapType=Jn,this.map=null,this.mapPass=null,this.matrix=new Xt,this.autoUpdate=!0,this.needsUpdate=!1,this._frustum=new gu,this._frameExtents=new $e(1,1),this._viewportCount=1,this._viewports=[new $t(0,0,1,1)]}getViewportCount(){return this._viewportCount}getFrustum(){return this._frustum}updateMatrices(e){const t=this.camera,n=this.matrix;lh.setFromMatrixPosition(e.matrixWorld),t.position.copy(lh),ch.setFromMatrixPosition(e.target.matrixWorld),t.lookAt(ch),t.updateMatrixWorld(),Il.multiplyMatrices(t.projectionMatrix,t.matrixWorldInverse),this._frustum.setFromProjectionMatrix(Il,t.coordinateSystem,t.reversedDepth),t.coordinateSystem===va||t.reversedDepth?n.set(.5,0,0,.5,0,.5,0,.5,0,0,1,0,0,0,0,1):n.set(.5,0,0,.5,0,.5,0,.5,0,0,.5,.5,0,0,0,1),n.multiply(Il)}getViewport(e){return this._viewports[e]}getFrameExtents(){return this._frameExtents}dispose(){this.map&&this.map.dispose(),this.mapPass&&this.mapPass.dispose()}copy(e){return this.camera=e.camera.clone(),this.intensity=e.intensity,this.bias=e.bias,this.radius=e.radius,this.autoUpdate=e.autoUpdate,this.needsUpdate=e.needsUpdate,this.normalBias=e.normalBias,this.blurSamples=e.blurSamples,this.mapSize.copy(e.mapSize),this.biasNode=e.biasNode,this}clone(){return new this.constructor().copy(this)}toJSON(){const e={};return this.intensity!==1&&(e.intensity=this.intensity),this.bias!==0&&(e.bias=this.bias),this.normalBias!==0&&(e.normalBias=this.normalBias),this.radius!==1&&(e.radius=this.radius),(this.mapSize.x!==512||this.mapSize.y!==512)&&(e.mapSize=this.mapSize.toArray()),e.camera=this.camera.toJSON(!1).object,delete e.camera.matrix,e}}const eo=new X,to=new vr,Ai=new X;class sp extends _n{constructor(){super(),this.isCamera=!0,this.type="Camera",this.matrixWorldInverse=new Xt,this.projectionMatrix=new Xt,this.projectionMatrixInverse=new Xt,this.coordinateSystem=Ii,this._reversedDepth=!1}get reversedDepth(){return this._reversedDepth}copy(e,t){return super.copy(e,t),this.matrixWorldInverse.copy(e.matrixWorldInverse),this.projectionMatrix.copy(e.projectionMatrix),this.projectionMatrixInverse.copy(e.projectionMatrixInverse),this.coordinateSystem=e.coordinateSystem,this}getWorldDirection(e){return super.getWorldDirection(e).negate()}updateMatrixWorld(e){super.updateMatrixWorld(e),this.matrixWorld.decompose(eo,to,Ai),Ai.x===1&&Ai.y===1&&Ai.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(eo,to,Ai.set(1,1,1)).invert()}updateWorldMatrix(e,t,n=!1){super.updateWorldMatrix(e,t,n),this.matrixWorld.decompose(eo,to,Ai),Ai.x===1&&Ai.y===1&&Ai.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(eo,to,Ai.set(1,1,1)).invert()}clone(){return new this.constructor().copy(this)}}const ur=new X,uh=new $e,fh=new $e;class oi extends sp{constructor(e=50,t=1,n=.1,r=2e3){super(),this.isPerspectiveCamera=!0,this.type="PerspectiveCamera",this.fov=e,this.zoom=1,this.near=n,this.far=r,this.focus=10,this.aspect=t,this.view=null,this.filmGauge=35,this.filmOffset=0,this.updateProjectionMatrix()}copy(e,t){return super.copy(e,t),this.fov=e.fov,this.zoom=e.zoom,this.near=e.near,this.far=e.far,this.focus=e.focus,this.aspect=e.aspect,this.view=e.view===null?null:Object.assign({},e.view),this.filmGauge=e.filmGauge,this.filmOffset=e.filmOffset,this}setFocalLength(e){const t=.5*this.getFilmHeight()/e;this.fov=Vc*2*Math.atan(t),this.updateProjectionMatrix()}getFocalLength(){const e=Math.tan(ho*.5*this.fov);return .5*this.getFilmHeight()/e}getEffectiveFOV(){return Vc*2*Math.atan(Math.tan(ho*.5*this.fov)/this.zoom)}getFilmWidth(){return this.filmGauge*Math.min(this.aspect,1)}getFilmHeight(){return this.filmGauge/Math.max(this.aspect,1)}getViewBounds(e,t,n){ur.set(-1,-1,.5).applyMatrix4(this.projectionMatrixInverse),t.set(ur.x,ur.y).multiplyScalar(-e/ur.z),ur.set(1,1,.5).applyMatrix4(this.projectionMatrixInverse),n.set(ur.x,ur.y).multiplyScalar(-e/ur.z)}getViewSize(e,t){return this.getViewBounds(e,uh,fh),t.subVectors(fh,uh)}setViewOffset(e,t,n,r,s,a){this.aspect=e/t,this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=e,this.view.fullHeight=t,this.view.offsetX=n,this.view.offsetY=r,this.view.width=s,this.view.height=a,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){const e=this.near;let t=e*Math.tan(ho*.5*this.fov)/this.zoom,n=2*t,r=this.aspect*n,s=-.5*r;const a=this.view;if(this.view!==null&&this.view.enabled){const l=a.fullWidth,c=a.fullHeight;s+=a.offsetX*r/l,t-=a.offsetY*n/c,r*=a.width/l,n*=a.height/c}const o=this.filmOffset;o!==0&&(s+=e*o/this.getFilmWidth()),this.projectionMatrix.makePerspective(s,s+r,t,t-n,e,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(e){const t=super.toJSON(e);return t.object.fov=this.fov,t.object.zoom=this.zoom,t.object.near=this.near,t.object.far=this.far,t.object.focus=this.focus,t.object.aspect=this.aspect,this.view!==null&&(t.object.view=Object.assign({},this.view)),t.object.filmGauge=this.filmGauge,t.object.filmOffset=this.filmOffset,t}}class _u extends sp{constructor(e=-1,t=1,n=1,r=-1,s=.1,a=2e3){super(),this.isOrthographicCamera=!0,this.type="OrthographicCamera",this.zoom=1,this.view=null,this.left=e,this.right=t,this.top=n,this.bottom=r,this.near=s,this.far=a,this.updateProjectionMatrix()}copy(e,t){return super.copy(e,t),this.left=e.left,this.right=e.right,this.top=e.top,this.bottom=e.bottom,this.near=e.near,this.far=e.far,this.zoom=e.zoom,this.view=e.view===null?null:Object.assign({},e.view),this}setViewOffset(e,t,n,r,s,a){this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=e,this.view.fullHeight=t,this.view.offsetX=n,this.view.offsetY=r,this.view.width=s,this.view.height=a,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){const e=(this.right-this.left)/(2*this.zoom),t=(this.top-this.bottom)/(2*this.zoom),n=(this.right+this.left)/2,r=(this.top+this.bottom)/2;let s=n-e,a=n+e,o=r+t,l=r-t;if(this.view!==null&&this.view.enabled){const c=(this.right-this.left)/this.view.fullWidth/this.zoom,f=(this.top-this.bottom)/this.view.fullHeight/this.zoom;s+=c*this.view.offsetX,a=s+c*this.view.width,o-=f*this.view.offsetY,l=o-f*this.view.height}this.projectionMatrix.makeOrthographic(s,a,o,l,this.near,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(e){const t=super.toJSON(e);return t.object.zoom=this.zoom,t.object.left=this.left,t.object.right=this.right,t.object.top=this.top,t.object.bottom=this.bottom,t.object.near=this.near,t.object.far=this.far,this.view!==null&&(t.object.view=Object.assign({},this.view)),t}}class c_ extends l_{constructor(){super(new _u(-5,5,5,-5,.5,500)),this.isDirectionalLightShadow=!0}}class hh extends rp{constructor(e,t){super(e,t),this.isDirectionalLight=!0,this.type="DirectionalLight",this.position.copy(_n.DEFAULT_UP),this.updateMatrix(),this.target=new _n,this.shadow=new c_}dispose(){super.dispose(),this.shadow.dispose()}copy(e){return super.copy(e),this.target=e.target.clone(),this.shadow=e.shadow.clone(),this}toJSON(e){const t=super.toJSON(e);return t.object.shadow=this.shadow.toJSON(),t.object.target=this.target.uuid,t}}class u_ extends rp{constructor(e,t){super(e,t),this.isAmbientLight=!0,this.type="AmbientLight"}}const _s=-90,xs=1;class f_ extends _n{constructor(e,t,n){super(),this.type="CubeCamera",this.renderTarget=n,this.coordinateSystem=null,this.activeMipmapLevel=0;const r=new oi(_s,xs,e,t);r.layers=this.layers,this.add(r);const s=new oi(_s,xs,e,t);s.layers=this.layers,this.add(s);const a=new oi(_s,xs,e,t);a.layers=this.layers,this.add(a);const o=new oi(_s,xs,e,t);o.layers=this.layers,this.add(o);const l=new oi(_s,xs,e,t);l.layers=this.layers,this.add(l);const c=new oi(_s,xs,e,t);c.layers=this.layers,this.add(c)}updateCoordinateSystem(){const e=this.coordinateSystem,t=this.children.concat(),[n,r,s,a,o,l]=t;for(const c of t)this.remove(c);if(e===Ii)n.up.set(0,1,0),n.lookAt(1,0,0),r.up.set(0,1,0),r.lookAt(-1,0,0),s.up.set(0,0,-1),s.lookAt(0,1,0),a.up.set(0,0,1),a.lookAt(0,-1,0),o.up.set(0,1,0),o.lookAt(0,0,1),l.up.set(0,1,0),l.lookAt(0,0,-1);else if(e===va)n.up.set(0,-1,0),n.lookAt(-1,0,0),r.up.set(0,-1,0),r.lookAt(1,0,0),s.up.set(0,0,1),s.lookAt(0,1,0),a.up.set(0,0,-1),a.lookAt(0,-1,0),o.up.set(0,-1,0),o.lookAt(0,0,1),l.up.set(0,-1,0),l.lookAt(0,0,-1);else throw new Error("THREE.CubeCamera.updateCoordinateSystem(): Invalid coordinate system: "+e);for(const c of t)this.add(c),c.updateMatrixWorld()}update(e,t){this.parent===null&&this.updateMatrixWorld();const{renderTarget:n,activeMipmapLevel:r}=this;this.coordinateSystem!==e.coordinateSystem&&(this.coordinateSystem=e.coordinateSystem,this.updateCoordinateSystem());const[s,a,o,l,c,f]=this.children,h=e.getRenderTarget(),u=e.getActiveCubeFace(),d=e.getActiveMipmapLevel(),x=e.xr.enabled;e.xr.enabled=!1;const E=n.texture.generateMipmaps;n.texture.generateMipmaps=!1;let m=!1;e.isWebGLRenderer===!0?m=e.state.buffers.depth.getReversed():m=e.reversedDepthBuffer,e.setRenderTarget(n,0,r),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,s),e.setRenderTarget(n,1,r),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,a),e.setRenderTarget(n,2,r),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,o),e.setRenderTarget(n,3,r),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,l),e.setRenderTarget(n,4,r),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,c),n.texture.generateMipmaps=E,e.setRenderTarget(n,5,r),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,f),e.setRenderTarget(h,u,d),e.xr.enabled=x,n.texture.needsPMREMUpdate=!0}}class h_ extends oi{constructor(e=[]){super(),this.isArrayCamera=!0,this.isMultiViewCamera=!1,this.cameras=e}}class dh{constructor(e=1,t=0,n=0){this.radius=e,this.phi=t,this.theta=n}set(e,t,n){return this.radius=e,this.phi=t,this.theta=n,this}copy(e){return this.radius=e.radius,this.phi=e.phi,this.theta=e.theta,this}makeSafe(){return this.phi=ut(this.phi,1e-6,Math.PI-1e-6),this}setFromVector3(e){return this.setFromCartesianCoords(e.x,e.y,e.z)}setFromCartesianCoords(e,t,n){return this.radius=Math.sqrt(e*e+t*t+n*n),this.radius===0?(this.theta=0,this.phi=0):(this.theta=Math.atan2(e,n),this.phi=Math.acos(ut(t/this.radius,-1,1))),this}clone(){return new this.constructor().copy(this)}}const Eu=class Eu{constructor(e,t,n,r){this.elements=[1,0,0,1],e!==void 0&&this.set(e,t,n,r)}identity(){return this.set(1,0,0,1),this}fromArray(e,t=0){for(let n=0;n<4;n++)this.elements[n]=e[n+t];return this}set(e,t,n,r){const s=this.elements;return s[0]=e,s[2]=t,s[1]=n,s[3]=r,this}};Eu.prototype.isMatrix2=!0;let ph=Eu;class d_ extends ep{constructor(e=10,t=10,n=4473924,r=8947848){n=new ft(n),r=new ft(r);const s=t/2,a=e/t,o=e/2,l=[],c=[];for(let u=0,d=0,x=-o;u<=t;u++,x+=a){l.push(-o,0,x,o,0,x),l.push(x,0,-o,x,0,o);const E=u===s?n:r;E.toArray(c,d),d+=3,E.toArray(c,d),d+=3,E.toArray(c,d),d+=3,E.toArray(c,d),d+=3}const f=new gn;f.setAttribute("position",new Bn(l,3)),f.setAttribute("color",new Bn(c,3));const h=new zo({vertexColors:!0,toneMapped:!1});super(f,h),this.type="GridHelper"}dispose(){this.geometry.dispose(),this.material.dispose()}}class p_ extends ep{constructor(e=1){const t=[0,0,0,e,0,0,0,0,0,0,e,0,0,0,0,0,0,e],n=[1,0,0,1,.6,0,0,1,0,.6,1,0,0,0,1,0,.6,1],r=new gn;r.setAttribute("position",new Bn(t,3)),r.setAttribute("color",new Bn(n,3));const s=new zo({vertexColors:!0,toneMapped:!1});super(r,s),this.type="AxesHelper"}setColors(e,t,n){const r=new ft,s=this.geometry.attributes.color.array;return r.set(e),r.toArray(s,0),r.toArray(s,3),r.set(t),r.toArray(s,6),r.toArray(s,9),r.set(n),r.toArray(s,12),r.toArray(s,15),this.geometry.attributes.color.needsUpdate=!0,this}dispose(){this.geometry.dispose(),this.material.dispose()}}class m_ extends Sr{constructor(e,t=null){super(),this.object=e,this.domElement=t,this.enabled=!0,this.state=-1,this.keys={},this.mouseButtons={LEFT:null,MIDDLE:null,RIGHT:null},this.touches={ONE:null,TWO:null}}connect(e){if(e===void 0){Je("Controls: connect() now requires an element.");return}this.domElement!==null&&this.disconnect(),this.domElement=e}disconnect(){}dispose(){}update(){}}function mh(i,e,t,n){const r=g_(n);switch(t){case Hd:return i*e;case Xd:return i*e/r.components*r.byteLength;case lu:return i*e/r.components*r.byteLength;case Vr:return i*e*2/r.components*r.byteLength;case cu:return i*e*2/r.components*r.byteLength;case Wd:return i*e*3/r.components*r.byteLength;case xi:return i*e*4/r.components*r.byteLength;case uu:return i*e*4/r.components*r.byteLength;case lo:case co:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*8;case uo:case fo:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*16;case fc:case dc:return Math.max(i,16)*Math.max(e,8)/4;case uc:case hc:return Math.max(i,8)*Math.max(e,8)/2;case pc:case mc:case _c:case xc:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*8;case gc:case vo:case vc:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*16;case Sc:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*16;case Mc:return Math.floor((i+4)/5)*Math.floor((e+3)/4)*16;case yc:return Math.floor((i+4)/5)*Math.floor((e+4)/5)*16;case Ec:return Math.floor((i+5)/6)*Math.floor((e+4)/5)*16;case bc:return Math.floor((i+5)/6)*Math.floor((e+5)/6)*16;case Tc:return Math.floor((i+7)/8)*Math.floor((e+4)/5)*16;case Ac:return Math.floor((i+7)/8)*Math.floor((e+5)/6)*16;case wc:return Math.floor((i+7)/8)*Math.floor((e+7)/8)*16;case Rc:return Math.floor((i+9)/10)*Math.floor((e+4)/5)*16;case Cc:return Math.floor((i+9)/10)*Math.floor((e+5)/6)*16;case Pc:return Math.floor((i+9)/10)*Math.floor((e+7)/8)*16;case Dc:return Math.floor((i+9)/10)*Math.floor((e+9)/10)*16;case Lc:return Math.floor((i+11)/12)*Math.floor((e+9)/10)*16;case Ic:return Math.floor((i+11)/12)*Math.floor((e+11)/12)*16;case Uc:case Nc:case Fc:return Math.ceil(i/4)*Math.ceil(e/4)*16;case Oc:case Bc:return Math.ceil(i/4)*Math.ceil(e/4)*8;case So:case zc:return Math.ceil(i/4)*Math.ceil(e/4)*16}throw new Error(`Unable to determine texture byte length for ${t} format.`)}function g_(i){switch(i){case Jn:case zd:return{byteLength:1,components:1};case _a:case kd:case Zi:return{byteLength:2,components:1};case au:case ou:return{byteLength:2,components:4};case Fi:case su:case Li:return{byteLength:4,components:1};case Vd:case Gd:return{byteLength:4,components:3}}throw new Error(`THREE.TextureUtils: Unknown texture type ${i}.`)}typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("register",{detail:{revision:ru}}));typeof window<"u"&&(window.__THREE__?Je("WARNING: Multiple instances of Three.js being imported."):window.__THREE__=ru);/**
 * @license
 * Copyright 2010-2026 Three.js Authors
 * SPDX-License-Identifier: MIT
 */function ap(){let i=null,e=!1,t=null,n=null;function r(s,a){t(s,a),n=i.requestAnimationFrame(r)}return{start:function(){e!==!0&&t!==null&&i!==null&&(n=i.requestAnimationFrame(r),e=!0)},stop:function(){i!==null&&i.cancelAnimationFrame(n),e=!1},setAnimationLoop:function(s){t=s},setContext:function(s){i=s}}}function __(i){const e=new WeakMap;function t(o,l){const c=o.array,f=o.usage,h=c.byteLength,u=i.createBuffer();i.bindBuffer(l,u),i.bufferData(l,c,f),o.onUploadCallback();let d;if(c instanceof Float32Array)d=i.FLOAT;else if(typeof Float16Array<"u"&&c instanceof Float16Array)d=i.HALF_FLOAT;else if(c instanceof Uint16Array)o.isFloat16BufferAttribute?d=i.HALF_FLOAT:d=i.UNSIGNED_SHORT;else if(c instanceof Int16Array)d=i.SHORT;else if(c instanceof Uint32Array)d=i.UNSIGNED_INT;else if(c instanceof Int32Array)d=i.INT;else if(c instanceof Int8Array)d=i.BYTE;else if(c instanceof Uint8Array)d=i.UNSIGNED_BYTE;else if(c instanceof Uint8ClampedArray)d=i.UNSIGNED_BYTE;else throw new Error("THREE.WebGLAttributes: Unsupported buffer data format: "+c);return{buffer:u,type:d,bytesPerElement:c.BYTES_PER_ELEMENT,version:o.version,size:h}}function n(o,l,c){const f=l.array,h=l.updateRanges;if(i.bindBuffer(c,o),h.length===0)i.bufferSubData(c,0,f);else{h.sort((d,x)=>d.start-x.start);let u=0;for(let d=1;d<h.length;d++){const x=h[u],E=h[d];E.start<=x.start+x.count+1?x.count=Math.max(x.count,E.start+E.count-x.start):(++u,h[u]=E)}h.length=u+1;for(let d=0,x=h.length;d<x;d++){const E=h[d];i.bufferSubData(c,E.start*f.BYTES_PER_ELEMENT,f,E.start,E.count)}l.clearUpdateRanges()}l.onUploadCallback()}function r(o){return o.isInterleavedBufferAttribute&&(o=o.data),e.get(o)}function s(o){o.isInterleavedBufferAttribute&&(o=o.data);const l=e.get(o);l&&(i.deleteBuffer(l.buffer),e.delete(o))}function a(o,l){if(o.isInterleavedBufferAttribute&&(o=o.data),o.isGLBufferAttribute){const f=e.get(o);(!f||f.version<o.version)&&e.set(o,{buffer:o.buffer,type:o.type,bytesPerElement:o.elementSize,version:o.version});return}const c=e.get(o);if(c===void 0)e.set(o,t(o,l));else if(c.version<o.version){if(c.size!==o.array.byteLength)throw new Error("THREE.WebGLAttributes: The size of the buffer attribute's array buffer does not match the original size. Resizing buffer attributes is not supported.");n(c.buffer,o,l),c.version=o.version}}return{get:r,remove:s,update:a}}var x_=`#ifdef USE_ALPHAHASH
	if ( diffuseColor.a < getAlphaHashThreshold( vPosition ) ) discard;
#endif`,v_=`#ifdef USE_ALPHAHASH
	const float ALPHA_HASH_SCALE = 0.05;
	float hash2D( vec2 value ) {
		return fract( 1.0e4 * sin( 17.0 * value.x + 0.1 * value.y ) * ( 0.1 + abs( sin( 13.0 * value.y + value.x ) ) ) );
	}
	float hash3D( vec3 value ) {
		return hash2D( vec2( hash2D( value.xy ), value.z ) );
	}
	float getAlphaHashThreshold( vec3 position ) {
		float maxDeriv = max(
			length( dFdx( position.xyz ) ),
			length( dFdy( position.xyz ) )
		);
		float pixScale = 1.0 / ( ALPHA_HASH_SCALE * maxDeriv );
		vec2 pixScales = vec2(
			exp2( floor( log2( pixScale ) ) ),
			exp2( ceil( log2( pixScale ) ) )
		);
		vec2 alpha = vec2(
			hash3D( floor( pixScales.x * position.xyz ) ),
			hash3D( floor( pixScales.y * position.xyz ) )
		);
		float lerpFactor = fract( log2( pixScale ) );
		float x = ( 1.0 - lerpFactor ) * alpha.x + lerpFactor * alpha.y;
		float a = min( lerpFactor, 1.0 - lerpFactor );
		vec3 cases = vec3(
			x * x / ( 2.0 * a * ( 1.0 - a ) ),
			( x - 0.5 * a ) / ( 1.0 - a ),
			1.0 - ( ( 1.0 - x ) * ( 1.0 - x ) / ( 2.0 * a * ( 1.0 - a ) ) )
		);
		float threshold = ( x < ( 1.0 - a ) )
			? ( ( x < a ) ? cases.x : cases.y )
			: cases.z;
		return clamp( threshold , 1.0e-6, 1.0 );
	}
#endif`,S_=`#ifdef USE_ALPHAMAP
	diffuseColor.a *= texture2D( alphaMap, vAlphaMapUv ).g;
#endif`,M_=`#ifdef USE_ALPHAMAP
	uniform sampler2D alphaMap;
#endif`,y_=`#ifdef USE_ALPHATEST
	#ifdef ALPHA_TO_COVERAGE
	diffuseColor.a = smoothstep( alphaTest, alphaTest + fwidth( diffuseColor.a ), diffuseColor.a );
	if ( diffuseColor.a == 0.0 ) discard;
	#else
	if ( diffuseColor.a < alphaTest ) discard;
	#endif
#endif`,E_=`#ifdef USE_ALPHATEST
	uniform float alphaTest;
#endif`,b_=`#ifdef USE_AOMAP
	float ambientOcclusion = ( texture2D( aoMap, vAoMapUv ).r - 1.0 ) * aoMapIntensity + 1.0;
	reflectedLight.indirectDiffuse *= ambientOcclusion;
	#if defined( USE_CLEARCOAT ) 
		clearcoatSpecularIndirect *= ambientOcclusion;
	#endif
	#if defined( USE_SHEEN ) 
		sheenSpecularIndirect *= ambientOcclusion;
	#endif
	#if defined( USE_ENVMAP ) && defined( STANDARD )
		float dotNV = saturate( dot( geometryNormal, geometryViewDir ) );
		reflectedLight.indirectSpecular *= computeSpecularOcclusion( dotNV, ambientOcclusion, material.roughness );
	#endif
#endif`,T_=`#ifdef USE_AOMAP
	uniform sampler2D aoMap;
	uniform float aoMapIntensity;
#endif`,A_=`#ifdef USE_BATCHING
	#if ! defined( GL_ANGLE_multi_draw )
	#define gl_DrawID _gl_DrawID
	uniform int _gl_DrawID;
	#endif
	uniform highp sampler2D batchingTexture;
	uniform highp usampler2D batchingIdTexture;
	mat4 getBatchingMatrix( const in float i ) {
		int size = textureSize( batchingTexture, 0 ).x;
		int j = int( i ) * 4;
		int x = j % size;
		int y = j / size;
		vec4 v1 = texelFetch( batchingTexture, ivec2( x, y ), 0 );
		vec4 v2 = texelFetch( batchingTexture, ivec2( x + 1, y ), 0 );
		vec4 v3 = texelFetch( batchingTexture, ivec2( x + 2, y ), 0 );
		vec4 v4 = texelFetch( batchingTexture, ivec2( x + 3, y ), 0 );
		return mat4( v1, v2, v3, v4 );
	}
	float getIndirectIndex( const in int i ) {
		int size = textureSize( batchingIdTexture, 0 ).x;
		int x = i % size;
		int y = i / size;
		return float( texelFetch( batchingIdTexture, ivec2( x, y ), 0 ).r );
	}
#endif
#ifdef USE_BATCHING_COLOR
	uniform sampler2D batchingColorTexture;
	vec4 getBatchingColor( const in float i ) {
		int size = textureSize( batchingColorTexture, 0 ).x;
		int j = int( i );
		int x = j % size;
		int y = j / size;
		return texelFetch( batchingColorTexture, ivec2( x, y ), 0 );
	}
#endif`,w_=`#ifdef USE_BATCHING
	mat4 batchingMatrix = getBatchingMatrix( getIndirectIndex( gl_DrawID ) );
#endif`,R_=`vec3 transformed = vec3( position );
#ifdef USE_ALPHAHASH
	vPosition = vec3( position );
#endif`,C_=`vec3 objectNormal = vec3( normal );
#ifdef USE_TANGENT
	vec3 objectTangent = vec3( tangent.xyz );
#endif`,P_=`float G_BlinnPhong_Implicit( ) {
	return 0.25;
}
float D_BlinnPhong( const in float shininess, const in float dotNH ) {
	return RECIPROCAL_PI * ( shininess * 0.5 + 1.0 ) * pow( dotNH, shininess );
}
vec3 BRDF_BlinnPhong( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in vec3 specularColor, const in float shininess ) {
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNH = saturate( dot( normal, halfDir ) );
	float dotVH = saturate( dot( viewDir, halfDir ) );
	vec3 F = F_Schlick( specularColor, 1.0, dotVH );
	float G = G_BlinnPhong_Implicit( );
	float D = D_BlinnPhong( shininess, dotNH );
	return F * ( G * D );
} // validated`,D_=`#ifdef USE_IRIDESCENCE
	const mat3 XYZ_TO_REC709 = mat3(
		 3.2404542, -0.9692660,  0.0556434,
		-1.5371385,  1.8760108, -0.2040259,
		-0.4985314,  0.0415560,  1.0572252
	);
	vec3 Fresnel0ToIor( vec3 fresnel0 ) {
		vec3 sqrtF0 = sqrt( fresnel0 );
		return ( vec3( 1.0 ) + sqrtF0 ) / ( vec3( 1.0 ) - sqrtF0 );
	}
	vec3 IorToFresnel0( vec3 transmittedIor, float incidentIor ) {
		return pow2( ( transmittedIor - vec3( incidentIor ) ) / ( transmittedIor + vec3( incidentIor ) ) );
	}
	float IorToFresnel0( float transmittedIor, float incidentIor ) {
		return pow2( ( transmittedIor - incidentIor ) / ( transmittedIor + incidentIor ));
	}
	vec3 evalSensitivity( float OPD, vec3 shift ) {
		float phase = 2.0 * PI * OPD * 1.0e-9;
		vec3 val = vec3( 5.4856e-13, 4.4201e-13, 5.2481e-13 );
		vec3 pos = vec3( 1.6810e+06, 1.7953e+06, 2.2084e+06 );
		vec3 var = vec3( 4.3278e+09, 9.3046e+09, 6.6121e+09 );
		vec3 xyz = val * sqrt( 2.0 * PI * var ) * cos( pos * phase + shift ) * exp( - pow2( phase ) * var );
		xyz.x += 9.7470e-14 * sqrt( 2.0 * PI * 4.5282e+09 ) * cos( 2.2399e+06 * phase + shift[ 0 ] ) * exp( - 4.5282e+09 * pow2( phase ) );
		xyz /= 1.0685e-7;
		vec3 rgb = XYZ_TO_REC709 * xyz;
		return rgb;
	}
	vec3 evalIridescence( float outsideIOR, float eta2, float cosTheta1, float thinFilmThickness, vec3 baseF0 ) {
		vec3 I;
		float iridescenceIOR = mix( outsideIOR, eta2, smoothstep( 0.0, 0.03, thinFilmThickness ) );
		float sinTheta2Sq = pow2( outsideIOR / iridescenceIOR ) * ( 1.0 - pow2( cosTheta1 ) );
		float cosTheta2Sq = 1.0 - sinTheta2Sq;
		if ( cosTheta2Sq < 0.0 ) {
			return vec3( 1.0 );
		}
		float cosTheta2 = sqrt( cosTheta2Sq );
		float R0 = IorToFresnel0( iridescenceIOR, outsideIOR );
		float R12 = F_Schlick( R0, 1.0, cosTheta1 );
		float T121 = 1.0 - R12;
		float phi12 = 0.0;
		if ( iridescenceIOR < outsideIOR ) phi12 = PI;
		float phi21 = PI - phi12;
		vec3 baseIOR = Fresnel0ToIor( clamp( baseF0, 0.0, 0.9999 ) );		vec3 R1 = IorToFresnel0( baseIOR, iridescenceIOR );
		vec3 R23 = F_Schlick( R1, 1.0, cosTheta2 );
		vec3 phi23 = vec3( 0.0 );
		if ( baseIOR[ 0 ] < iridescenceIOR ) phi23[ 0 ] = PI;
		if ( baseIOR[ 1 ] < iridescenceIOR ) phi23[ 1 ] = PI;
		if ( baseIOR[ 2 ] < iridescenceIOR ) phi23[ 2 ] = PI;
		float OPD = 2.0 * iridescenceIOR * thinFilmThickness * cosTheta2;
		vec3 phi = vec3( phi21 ) + phi23;
		vec3 R123 = clamp( R12 * R23, 1e-5, 0.9999 );
		vec3 r123 = sqrt( R123 );
		vec3 Rs = pow2( T121 ) * R23 / ( vec3( 1.0 ) - R123 );
		vec3 C0 = R12 + Rs;
		I = C0;
		vec3 Cm = Rs - T121;
		for ( int m = 1; m <= 2; ++ m ) {
			Cm *= r123;
			vec3 Sm = 2.0 * evalSensitivity( float( m ) * OPD, float( m ) * phi );
			I += Cm * Sm;
		}
		return max( I, vec3( 0.0 ) );
	}
#endif`,L_=`#ifdef USE_BUMPMAP
	uniform sampler2D bumpMap;
	uniform float bumpScale;
	vec2 dHdxy_fwd() {
		vec2 dSTdx = dFdx( vBumpMapUv );
		vec2 dSTdy = dFdy( vBumpMapUv );
		float Hll = bumpScale * texture2D( bumpMap, vBumpMapUv ).x;
		float dBx = bumpScale * texture2D( bumpMap, vBumpMapUv + dSTdx ).x - Hll;
		float dBy = bumpScale * texture2D( bumpMap, vBumpMapUv + dSTdy ).x - Hll;
		return vec2( dBx, dBy );
	}
	vec3 perturbNormalArb( vec3 surf_pos, vec3 surf_norm, vec2 dHdxy, float faceDirection ) {
		vec3 vSigmaX = normalize( dFdx( surf_pos.xyz ) );
		vec3 vSigmaY = normalize( dFdy( surf_pos.xyz ) );
		vec3 vN = surf_norm;
		vec3 R1 = cross( vSigmaY, vN );
		vec3 R2 = cross( vN, vSigmaX );
		float fDet = dot( vSigmaX, R1 ) * faceDirection;
		vec3 vGrad = sign( fDet ) * ( dHdxy.x * R1 + dHdxy.y * R2 );
		return normalize( abs( fDet ) * surf_norm - vGrad );
	}
#endif`,I_=`#if NUM_CLIPPING_PLANES > 0
	vec4 plane;
	#ifdef ALPHA_TO_COVERAGE
		float distanceToPlane, distanceGradient;
		float clipOpacity = 1.0;
		#pragma unroll_loop_start
		for ( int i = 0; i < UNION_CLIPPING_PLANES; i ++ ) {
			plane = clippingPlanes[ i ];
			distanceToPlane = - dot( vClipPosition, plane.xyz ) + plane.w;
			distanceGradient = fwidth( distanceToPlane ) / 2.0;
			clipOpacity *= smoothstep( - distanceGradient, distanceGradient, distanceToPlane );
			if ( clipOpacity == 0.0 ) discard;
		}
		#pragma unroll_loop_end
		#if UNION_CLIPPING_PLANES < NUM_CLIPPING_PLANES
			float unionClipOpacity = 1.0;
			#pragma unroll_loop_start
			for ( int i = UNION_CLIPPING_PLANES; i < NUM_CLIPPING_PLANES; i ++ ) {
				plane = clippingPlanes[ i ];
				distanceToPlane = - dot( vClipPosition, plane.xyz ) + plane.w;
				distanceGradient = fwidth( distanceToPlane ) / 2.0;
				unionClipOpacity *= 1.0 - smoothstep( - distanceGradient, distanceGradient, distanceToPlane );
			}
			#pragma unroll_loop_end
			clipOpacity *= 1.0 - unionClipOpacity;
		#endif
		diffuseColor.a *= clipOpacity;
		if ( diffuseColor.a == 0.0 ) discard;
	#else
		#pragma unroll_loop_start
		for ( int i = 0; i < UNION_CLIPPING_PLANES; i ++ ) {
			plane = clippingPlanes[ i ];
			if ( dot( vClipPosition, plane.xyz ) > plane.w ) discard;
		}
		#pragma unroll_loop_end
		#if UNION_CLIPPING_PLANES < NUM_CLIPPING_PLANES
			bool clipped = true;
			#pragma unroll_loop_start
			for ( int i = UNION_CLIPPING_PLANES; i < NUM_CLIPPING_PLANES; i ++ ) {
				plane = clippingPlanes[ i ];
				clipped = ( dot( vClipPosition, plane.xyz ) > plane.w ) && clipped;
			}
			#pragma unroll_loop_end
			if ( clipped ) discard;
		#endif
	#endif
#endif`,U_=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
	uniform vec4 clippingPlanes[ NUM_CLIPPING_PLANES ];
#endif`,N_=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
#endif`,F_=`#if NUM_CLIPPING_PLANES > 0
	vClipPosition = - mvPosition.xyz;
#endif`,O_=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	diffuseColor *= vColor;
#endif`,B_=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	varying vec4 vColor;
#endif`,z_=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
	varying vec4 vColor;
#endif`,k_=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
	vColor = vec4( 1.0 );
#endif
#ifdef USE_COLOR_ALPHA
	vColor *= color;
#elif defined( USE_COLOR )
	vColor.rgb *= color;
#endif
#ifdef USE_INSTANCING_COLOR
	vColor.rgb *= instanceColor.rgb;
#endif
#ifdef USE_BATCHING_COLOR
	vColor *= getBatchingColor( getIndirectIndex( gl_DrawID ) );
#endif`,V_=`#define PI 3.141592653589793
#define PI2 6.283185307179586
#define PI_HALF 1.5707963267948966
#define RECIPROCAL_PI 0.3183098861837907
#define RECIPROCAL_PI2 0.15915494309189535
#define EPSILON 1e-6
#ifndef saturate
#define saturate( a ) clamp( a, 0.0, 1.0 )
#endif
#define whiteComplement( a ) ( 1.0 - saturate( a ) )
float pow2( const in float x ) { return x*x; }
vec3 pow2( const in vec3 x ) { return x*x; }
float pow3( const in float x ) { return x*x*x; }
float pow4( const in float x ) { float x2 = x*x; return x2*x2; }
float max3( const in vec3 v ) { return max( max( v.x, v.y ), v.z ); }
float average( const in vec3 v ) { return dot( v, vec3( 0.3333333 ) ); }
highp float rand( const in vec2 uv ) {
	const highp float a = 12.9898, b = 78.233, c = 43758.5453;
	highp float dt = dot( uv.xy, vec2( a,b ) ), sn = mod( dt, PI );
	return fract( sin( sn ) * c );
}
#ifdef HIGH_PRECISION
	float precisionSafeLength( vec3 v ) { return length( v ); }
#else
	float precisionSafeLength( vec3 v ) {
		float maxComponent = max3( abs( v ) );
		return length( v / maxComponent ) * maxComponent;
	}
#endif
struct IncidentLight {
	vec3 color;
	vec3 direction;
	bool visible;
};
struct ReflectedLight {
	vec3 directDiffuse;
	vec3 directSpecular;
	vec3 indirectDiffuse;
	vec3 indirectSpecular;
};
#ifdef USE_ALPHAHASH
	varying vec3 vPosition;
#endif
vec3 transformDirection( in vec3 dir, in mat4 matrix ) {
	return normalize( ( matrix * vec4( dir, 0.0 ) ).xyz );
}
#define inverseTransformDirection transformDirectionByInverseViewMatrix
vec3 transformNormalByInverseViewMatrix( in vec3 normal, in mat4 viewMatrix ) {
	return normalize( ( vec4( normal, 0.0 ) * viewMatrix ).xyz );
}
vec3 transformDirectionByInverseViewMatrix( in vec3 dir, in mat4 viewMatrix ) {
	return normalize( ( vec4( dir, 0.0 ) * viewMatrix ).xyz );
}
bool isPerspectiveMatrix( mat4 m ) {
	return m[ 2 ][ 3 ] == - 1.0;
}
vec2 equirectUv( in vec3 dir ) {
	float u = atan( dir.z, dir.x ) * RECIPROCAL_PI2 + 0.5;
	float v = asin( clamp( dir.y, - 1.0, 1.0 ) ) * RECIPROCAL_PI + 0.5;
	return vec2( u, v );
}
vec3 BRDF_Lambert( const in vec3 diffuseColor ) {
	return RECIPROCAL_PI * diffuseColor;
}
vec3 F_Schlick( const in vec3 f0, const in float f90, const in float dotVH ) {
	float fresnel = exp2( ( - 5.55473 * dotVH - 6.98316 ) * dotVH );
	return f0 * ( 1.0 - fresnel ) + ( f90 * fresnel );
}
float F_Schlick( const in float f0, const in float f90, const in float dotVH ) {
	float fresnel = exp2( ( - 5.55473 * dotVH - 6.98316 ) * dotVH );
	return f0 * ( 1.0 - fresnel ) + ( f90 * fresnel );
} // validated`,G_=`#ifdef ENVMAP_TYPE_CUBE_UV
	#define cubeUV_minMipLevel 4.0
	#define cubeUV_minTileSize 16.0
	float getFace( vec3 direction ) {
		vec3 absDirection = abs( direction );
		float face = - 1.0;
		if ( absDirection.x > absDirection.z ) {
			if ( absDirection.x > absDirection.y )
				face = direction.x > 0.0 ? 0.0 : 3.0;
			else
				face = direction.y > 0.0 ? 1.0 : 4.0;
		} else {
			if ( absDirection.z > absDirection.y )
				face = direction.z > 0.0 ? 2.0 : 5.0;
			else
				face = direction.y > 0.0 ? 1.0 : 4.0;
		}
		return face;
	}
	vec2 getUV( vec3 direction, float face ) {
		vec2 uv;
		if ( face == 0.0 ) {
			uv = vec2( direction.z, direction.y ) / abs( direction.x );
		} else if ( face == 1.0 ) {
			uv = vec2( - direction.x, - direction.z ) / abs( direction.y );
		} else if ( face == 2.0 ) {
			uv = vec2( - direction.x, direction.y ) / abs( direction.z );
		} else if ( face == 3.0 ) {
			uv = vec2( - direction.z, direction.y ) / abs( direction.x );
		} else if ( face == 4.0 ) {
			uv = vec2( - direction.x, direction.z ) / abs( direction.y );
		} else {
			uv = vec2( direction.x, direction.y ) / abs( direction.z );
		}
		return 0.5 * ( uv + 1.0 );
	}
	vec3 bilinearCubeUV( sampler2D envMap, vec3 direction, float mipInt ) {
		float face = getFace( direction );
		float filterInt = max( cubeUV_minMipLevel - mipInt, 0.0 );
		mipInt = max( mipInt, cubeUV_minMipLevel );
		float faceSize = exp2( mipInt );
		highp vec2 uv = getUV( direction, face ) * ( faceSize - 2.0 ) + 1.0;
		if ( face > 2.0 ) {
			uv.y += faceSize;
			face -= 3.0;
		}
		uv.x += face * faceSize;
		uv.x += filterInt * 3.0 * cubeUV_minTileSize;
		uv.y += 4.0 * ( exp2( CUBEUV_MAX_MIP ) - faceSize );
		uv.x *= CUBEUV_TEXEL_WIDTH;
		uv.y *= CUBEUV_TEXEL_HEIGHT;
		#ifdef texture2DGradEXT
			return texture2DGradEXT( envMap, uv, vec2( 0.0 ), vec2( 0.0 ) ).rgb;
		#else
			return texture2D( envMap, uv ).rgb;
		#endif
	}
	#define cubeUV_r0 1.0
	#define cubeUV_m0 - 2.0
	#define cubeUV_r1 0.8
	#define cubeUV_m1 - 1.0
	#define cubeUV_r4 0.4
	#define cubeUV_m4 2.0
	#define cubeUV_r5 0.305
	#define cubeUV_m5 3.0
	#define cubeUV_r6 0.21
	#define cubeUV_m6 4.0
	float roughnessToMip( float roughness ) {
		float mip = 0.0;
		if ( roughness >= cubeUV_r1 ) {
			mip = ( cubeUV_r0 - roughness ) * ( cubeUV_m1 - cubeUV_m0 ) / ( cubeUV_r0 - cubeUV_r1 ) + cubeUV_m0;
		} else if ( roughness >= cubeUV_r4 ) {
			mip = ( cubeUV_r1 - roughness ) * ( cubeUV_m4 - cubeUV_m1 ) / ( cubeUV_r1 - cubeUV_r4 ) + cubeUV_m1;
		} else if ( roughness >= cubeUV_r5 ) {
			mip = ( cubeUV_r4 - roughness ) * ( cubeUV_m5 - cubeUV_m4 ) / ( cubeUV_r4 - cubeUV_r5 ) + cubeUV_m4;
		} else if ( roughness >= cubeUV_r6 ) {
			mip = ( cubeUV_r5 - roughness ) * ( cubeUV_m6 - cubeUV_m5 ) / ( cubeUV_r5 - cubeUV_r6 ) + cubeUV_m5;
		} else {
			mip = - 2.0 * log2( 1.16 * roughness );		}
		return mip;
	}
	vec4 textureCubeUV( sampler2D envMap, vec3 sampleDir, float roughness ) {
		float mip = clamp( roughnessToMip( roughness ), cubeUV_m0, CUBEUV_MAX_MIP );
		float mipF = fract( mip );
		float mipInt = floor( mip );
		vec3 color0 = bilinearCubeUV( envMap, sampleDir, mipInt );
		if ( mipF == 0.0 ) {
			return vec4( color0, 1.0 );
		} else {
			vec3 color1 = bilinearCubeUV( envMap, sampleDir, mipInt + 1.0 );
			return vec4( mix( color0, color1, mipF ), 1.0 );
		}
	}
#endif`,H_=`vec3 transformedNormal = objectNormal;
#ifdef USE_TANGENT
	vec3 transformedTangent = objectTangent;
#endif
#ifdef USE_BATCHING
	mat3 bm = mat3( batchingMatrix );
	transformedNormal /= vec3( dot( bm[ 0 ], bm[ 0 ] ), dot( bm[ 1 ], bm[ 1 ] ), dot( bm[ 2 ], bm[ 2 ] ) );
	transformedNormal = bm * transformedNormal;
	#ifdef USE_TANGENT
		transformedTangent = bm * transformedTangent;
	#endif
#endif
#ifdef USE_INSTANCING
	mat3 im = mat3( instanceMatrix );
	transformedNormal /= vec3( dot( im[ 0 ], im[ 0 ] ), dot( im[ 1 ], im[ 1 ] ), dot( im[ 2 ], im[ 2 ] ) );
	transformedNormal = im * transformedNormal;
	#ifdef USE_TANGENT
		transformedTangent = im * transformedTangent;
	#endif
#endif
transformedNormal = normalMatrix * transformedNormal;
#ifdef FLIP_SIDED
	transformedNormal = - transformedNormal;
#endif
#ifdef USE_TANGENT
	transformedTangent = ( modelViewMatrix * vec4( transformedTangent, 0.0 ) ).xyz;
#endif`,W_=`#ifdef USE_DISPLACEMENTMAP
	uniform sampler2D displacementMap;
	uniform float displacementScale;
	uniform float displacementBias;
#endif`,X_=`#ifdef USE_DISPLACEMENTMAP
	transformed += normalize( objectNormal ) * ( texture2D( displacementMap, vDisplacementMapUv ).x * displacementScale + displacementBias );
#endif`,Y_=`#ifdef USE_EMISSIVEMAP
	vec4 emissiveColor = texture2D( emissiveMap, vEmissiveMapUv );
	#ifdef DECODE_VIDEO_TEXTURE_EMISSIVE
		emissiveColor = sRGBTransferEOTF( emissiveColor );
	#endif
	totalEmissiveRadiance *= emissiveColor.rgb;
#endif`,q_=`#ifdef USE_EMISSIVEMAP
	uniform sampler2D emissiveMap;
#endif`,K_="gl_FragColor = linearToOutputTexel( gl_FragColor );",$_=`vec4 LinearTransferOETF( in vec4 value ) {
	return value;
}
vec4 sRGBTransferEOTF( in vec4 value ) {
	return vec4( mix( pow( value.rgb * 0.9478672986 + vec3( 0.0521327014 ), vec3( 2.4 ) ), value.rgb * 0.0773993808, vec3( lessThanEqual( value.rgb, vec3( 0.04045 ) ) ) ), value.a );
}
vec4 sRGBTransferOETF( in vec4 value ) {
	return vec4( mix( pow( value.rgb, vec3( 0.41666 ) ) * 1.055 - vec3( 0.055 ), value.rgb * 12.92, vec3( lessThanEqual( value.rgb, vec3( 0.0031308 ) ) ) ), value.a );
}`,Z_=`#ifdef USE_ENVMAP
	#ifdef ENV_WORLDPOS
		vec3 cameraToFrag;
		if ( isOrthographic ) {
			cameraToFrag = normalize( vec3( - viewMatrix[ 0 ][ 2 ], - viewMatrix[ 1 ][ 2 ], - viewMatrix[ 2 ][ 2 ] ) );
		} else {
			cameraToFrag = normalize( vWorldPosition - cameraPosition );
		}
		vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
		#ifdef ENVMAP_MODE_REFLECTION
			vec3 reflectVec = reflect( cameraToFrag, worldNormal );
		#else
			vec3 reflectVec = refract( cameraToFrag, worldNormal, refractionRatio );
		#endif
	#else
		vec3 reflectVec = vReflect;
	#endif
	#ifdef ENVMAP_TYPE_CUBE
		vec4 envColor = textureCube( envMap, envMapRotation * reflectVec );
		#ifdef ENVMAP_BLENDING_MULTIPLY
			outgoingLight = mix( outgoingLight, outgoingLight * envColor.xyz, specularStrength * reflectivity );
		#elif defined( ENVMAP_BLENDING_MIX )
			outgoingLight = mix( outgoingLight, envColor.xyz, specularStrength * reflectivity );
		#elif defined( ENVMAP_BLENDING_ADD )
			outgoingLight += envColor.xyz * specularStrength * reflectivity;
		#endif
	#endif
#endif`,J_=`#ifdef USE_ENVMAP
	uniform float envMapIntensity;
	uniform mat3 envMapRotation;
	#ifdef ENVMAP_TYPE_CUBE
		uniform samplerCube envMap;
	#else
		uniform sampler2D envMap;
	#endif
#endif`,j_=`#ifdef USE_ENVMAP
	uniform float reflectivity;
	#if defined( USE_BUMPMAP ) || defined( USE_NORMALMAP ) || defined( PHONG ) || defined( LAMBERT )
		#define ENV_WORLDPOS
	#endif
	#ifdef ENV_WORLDPOS
		varying vec3 vWorldPosition;
		uniform float refractionRatio;
	#else
		varying vec3 vReflect;
	#endif
#endif`,Q_=`#ifdef USE_ENVMAP
	#if defined( USE_BUMPMAP ) || defined( USE_NORMALMAP ) || defined( PHONG ) || defined( LAMBERT )
		#define ENV_WORLDPOS
	#endif
	#ifdef ENV_WORLDPOS
		
		varying vec3 vWorldPosition;
	#else
		varying vec3 vReflect;
		uniform float refractionRatio;
	#endif
#endif`,ex=`#ifdef USE_ENVMAP
	#ifdef ENV_WORLDPOS
		vWorldPosition = worldPosition.xyz;
	#else
		vec3 cameraToVertex;
		if ( isOrthographic ) {
			cameraToVertex = normalize( vec3( - viewMatrix[ 0 ][ 2 ], - viewMatrix[ 1 ][ 2 ], - viewMatrix[ 2 ][ 2 ] ) );
		} else {
			cameraToVertex = normalize( worldPosition.xyz - cameraPosition );
		}
		vec3 worldNormal = transformNormalByInverseViewMatrix( transformedNormal, viewMatrix );
		#ifdef ENVMAP_MODE_REFLECTION
			vReflect = reflect( cameraToVertex, worldNormal );
		#else
			vReflect = refract( cameraToVertex, worldNormal, refractionRatio );
		#endif
	#endif
#endif`,tx=`#ifdef USE_FOG
	vFogDepth = - mvPosition.z;
#endif`,nx=`#ifdef USE_FOG
	varying float vFogDepth;
#endif`,ix=`#ifdef USE_FOG
	#ifdef FOG_EXP2
		float fogFactor = 1.0 - exp( - fogDensity * fogDensity * vFogDepth * vFogDepth );
	#else
		float fogFactor = smoothstep( fogNear, fogFar, vFogDepth );
	#endif
	gl_FragColor.rgb = mix( gl_FragColor.rgb, fogColor, fogFactor );
#endif`,rx=`#ifdef USE_FOG
	uniform vec3 fogColor;
	varying float vFogDepth;
	#ifdef FOG_EXP2
		uniform float fogDensity;
	#else
		uniform float fogNear;
		uniform float fogFar;
	#endif
#endif`,sx=`#ifdef USE_GRADIENTMAP
	uniform sampler2D gradientMap;
#endif
vec3 getGradientIrradiance( vec3 normal, vec3 lightDirection ) {
	float dotNL = dot( normal, lightDirection );
	vec2 coord = vec2( dotNL * 0.5 + 0.5, 0.0 );
	#ifdef USE_GRADIENTMAP
		return vec3( texture2D( gradientMap, coord ).r );
	#else
		vec2 fw = fwidth( coord ) * 0.5;
		return mix( vec3( 0.7 ), vec3( 1.0 ), smoothstep( 0.7 - fw.x, 0.7 + fw.x, coord.x ) );
	#endif
}`,ax=`#ifdef USE_LIGHTMAP
	uniform sampler2D lightMap;
	uniform float lightMapIntensity;
#endif`,ox=`LambertMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularStrength = specularStrength;`,lx=`varying vec3 vViewPosition;
struct LambertMaterial {
	vec3 diffuseColor;
	float specularStrength;
};
void RE_Direct_Lambert( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in LambertMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
void RE_IndirectDiffuse_Lambert( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in LambertMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_Lambert
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Lambert`,cx=`uniform bool receiveShadow;
uniform vec3 ambientLightColor;
#if defined( USE_LIGHT_PROBES )
	uniform vec3 lightProbe[ 9 ];
#endif
vec3 shGetIrradianceAt( in vec3 normal, in vec3 shCoefficients[ 9 ] ) {
	float x = normal.x, y = normal.y, z = normal.z;
	vec3 result = shCoefficients[ 0 ] * 0.886227;
	result += shCoefficients[ 1 ] * 2.0 * 0.511664 * y;
	result += shCoefficients[ 2 ] * 2.0 * 0.511664 * z;
	result += shCoefficients[ 3 ] * 2.0 * 0.511664 * x;
	result += shCoefficients[ 4 ] * 2.0 * 0.429043 * x * y;
	result += shCoefficients[ 5 ] * 2.0 * 0.429043 * y * z;
	result += shCoefficients[ 6 ] * ( 0.743125 * z * z - 0.247708 );
	result += shCoefficients[ 7 ] * 2.0 * 0.429043 * x * z;
	result += shCoefficients[ 8 ] * 0.429043 * ( x * x - y * y );
	return result;
}
vec3 getLightProbeIrradiance( const in vec3 lightProbe[ 9 ], const in vec3 normal ) {
	vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
	vec3 irradiance = shGetIrradianceAt( worldNormal, lightProbe );
	return irradiance;
}
vec3 getAmbientLightIrradiance( const in vec3 ambientLightColor ) {
	vec3 irradiance = ambientLightColor;
	return irradiance;
}
float getDistanceAttenuation( const in float lightDistance, const in float cutoffDistance, const in float decayExponent ) {
	float distanceFalloff = 1.0 / max( pow( lightDistance, decayExponent ), 0.01 );
	if ( cutoffDistance > 0.0 ) {
		distanceFalloff *= pow2( saturate( 1.0 - pow4( lightDistance / cutoffDistance ) ) );
	}
	return distanceFalloff;
}
float getSpotAttenuation( const in float coneCosine, const in float penumbraCosine, const in float angleCosine ) {
	return smoothstep( coneCosine, penumbraCosine, angleCosine );
}
#if NUM_DIR_LIGHTS > 0
	struct DirectionalLight {
		vec3 direction;
		vec3 color;
	};
	uniform DirectionalLight directionalLights[ NUM_DIR_LIGHTS ];
	void getDirectionalLightInfo( const in DirectionalLight directionalLight, out IncidentLight light ) {
		light.color = directionalLight.color;
		light.direction = directionalLight.direction;
		light.visible = true;
	}
#endif
#if NUM_POINT_LIGHTS > 0
	struct PointLight {
		vec3 position;
		vec3 color;
		float distance;
		float decay;
	};
	uniform PointLight pointLights[ NUM_POINT_LIGHTS ];
	void getPointLightInfo( const in PointLight pointLight, const in vec3 geometryPosition, out IncidentLight light ) {
		vec3 lVector = pointLight.position - geometryPosition;
		light.direction = normalize( lVector );
		float lightDistance = length( lVector );
		light.color = pointLight.color;
		light.color *= getDistanceAttenuation( lightDistance, pointLight.distance, pointLight.decay );
		light.visible = ( light.color != vec3( 0.0 ) );
	}
#endif
#if NUM_SPOT_LIGHTS > 0
	struct SpotLight {
		vec3 position;
		vec3 direction;
		vec3 color;
		float distance;
		float decay;
		float coneCos;
		float penumbraCos;
	};
	uniform SpotLight spotLights[ NUM_SPOT_LIGHTS ];
	void getSpotLightInfo( const in SpotLight spotLight, const in vec3 geometryPosition, out IncidentLight light ) {
		vec3 lVector = spotLight.position - geometryPosition;
		light.direction = normalize( lVector );
		float angleCos = dot( light.direction, spotLight.direction );
		float spotAttenuation = getSpotAttenuation( spotLight.coneCos, spotLight.penumbraCos, angleCos );
		if ( spotAttenuation > 0.0 ) {
			float lightDistance = length( lVector );
			light.color = spotLight.color * spotAttenuation;
			light.color *= getDistanceAttenuation( lightDistance, spotLight.distance, spotLight.decay );
			light.visible = ( light.color != vec3( 0.0 ) );
		} else {
			light.color = vec3( 0.0 );
			light.visible = false;
		}
	}
#endif
#if NUM_RECT_AREA_LIGHTS > 0
	struct RectAreaLight {
		vec3 color;
		vec3 position;
		vec3 halfWidth;
		vec3 halfHeight;
	};
	uniform sampler2D ltc_1;	uniform sampler2D ltc_2;
	uniform RectAreaLight rectAreaLights[ NUM_RECT_AREA_LIGHTS ];
#endif
#if NUM_HEMI_LIGHTS > 0
	struct HemisphereLight {
		vec3 direction;
		vec3 skyColor;
		vec3 groundColor;
	};
	uniform HemisphereLight hemisphereLights[ NUM_HEMI_LIGHTS ];
	vec3 getHemisphereLightIrradiance( const in HemisphereLight hemiLight, const in vec3 normal ) {
		float dotNL = dot( normal, hemiLight.direction );
		float hemiDiffuseWeight = 0.5 * dotNL + 0.5;
		vec3 irradiance = mix( hemiLight.groundColor, hemiLight.skyColor, hemiDiffuseWeight );
		return irradiance;
	}
#endif
#include <lightprobes_pars_fragment>`,ux=`#ifdef USE_ENVMAP
	vec3 getIBLIrradiance( const in vec3 normal ) {
		#ifdef ENVMAP_TYPE_CUBE_UV
			vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
			vec4 envMapColor = textureCubeUV( envMap, envMapRotation * worldNormal, 1.0 );
			return PI * envMapColor.rgb * envMapIntensity;
		#else
			return vec3( 0.0 );
		#endif
	}
	vec3 getIBLRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness ) {
		#ifdef ENVMAP_TYPE_CUBE_UV
			vec3 reflectVec = reflect( - viewDir, normal );
			reflectVec = normalize( mix( reflectVec, normal, pow4( roughness ) ) );
			reflectVec = transformDirectionByInverseViewMatrix( reflectVec, viewMatrix );
			vec4 envMapColor = textureCubeUV( envMap, envMapRotation * reflectVec, roughness );
			return envMapColor.rgb * envMapIntensity;
		#else
			return vec3( 0.0 );
		#endif
	}
	#ifdef USE_ANISOTROPY
		vec3 getIBLAnisotropyRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness, const in vec3 bitangent, const in float anisotropy ) {
			#ifdef ENVMAP_TYPE_CUBE_UV
				vec3 bentNormal = cross( bitangent, viewDir );
				bentNormal = normalize( cross( bentNormal, bitangent ) );
				bentNormal = normalize( mix( bentNormal, normal, pow2( pow2( 1.0 - anisotropy * ( 1.0 - roughness ) ) ) ) );
				return getIBLRadiance( viewDir, bentNormal, roughness );
			#else
				return vec3( 0.0 );
			#endif
		}
	#endif
#endif`,fx=`ToonMaterial material;
material.diffuseColor = diffuseColor.rgb;`,hx=`varying vec3 vViewPosition;
struct ToonMaterial {
	vec3 diffuseColor;
};
void RE_Direct_Toon( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in ToonMaterial material, inout ReflectedLight reflectedLight ) {
	vec3 irradiance = getGradientIrradiance( geometryNormal, directLight.direction ) * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
void RE_IndirectDiffuse_Toon( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in ToonMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_Toon
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Toon`,dx=`BlinnPhongMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularColor = specular;
material.specularShininess = shininess;
material.specularStrength = specularStrength;`,px=`varying vec3 vViewPosition;
struct BlinnPhongMaterial {
	vec3 diffuseColor;
	vec3 specularColor;
	float specularShininess;
	float specularStrength;
};
void RE_Direct_BlinnPhong( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in BlinnPhongMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
	reflectedLight.directSpecular += irradiance * BRDF_BlinnPhong( directLight.direction, geometryViewDir, geometryNormal, material.specularColor, material.specularShininess ) * material.specularStrength;
}
void RE_IndirectDiffuse_BlinnPhong( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in BlinnPhongMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_BlinnPhong
#define RE_IndirectDiffuse		RE_IndirectDiffuse_BlinnPhong`,mx=`PhysicalMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.diffuseContribution = diffuseColor.rgb * ( 1.0 - metalnessFactor );
material.metalness = metalnessFactor;
vec3 dxy = max( abs( dFdx( nonPerturbedNormal ) ), abs( dFdy( nonPerturbedNormal ) ) );
float geometryRoughness = max( max( dxy.x, dxy.y ), dxy.z );
material.roughness = max( roughnessFactor, 0.0525 );material.roughness += geometryRoughness;
material.roughness = min( material.roughness, 1.0 );
#ifdef IOR
	material.ior = ior;
	#ifdef USE_SPECULAR
		float specularIntensityFactor = specularIntensity;
		vec3 specularColorFactor = specularColor;
		#ifdef USE_SPECULAR_COLORMAP
			specularColorFactor *= texture2D( specularColorMap, vSpecularColorMapUv ).rgb;
		#endif
		#ifdef USE_SPECULAR_INTENSITYMAP
			specularIntensityFactor *= texture2D( specularIntensityMap, vSpecularIntensityMapUv ).a;
		#endif
		material.specularF90 = mix( specularIntensityFactor, 1.0, metalnessFactor );
	#else
		float specularIntensityFactor = 1.0;
		vec3 specularColorFactor = vec3( 1.0 );
		material.specularF90 = 1.0;
	#endif
	material.specularColor = min( pow2( ( material.ior - 1.0 ) / ( material.ior + 1.0 ) ) * specularColorFactor, vec3( 1.0 ) ) * specularIntensityFactor;
	material.specularColorBlended = mix( material.specularColor, diffuseColor.rgb, metalnessFactor );
#else
	material.specularColor = vec3( 0.04 );
	material.specularColorBlended = mix( material.specularColor, diffuseColor.rgb, metalnessFactor );
	material.specularF90 = 1.0;
#endif
#ifdef USE_CLEARCOAT
	material.clearcoat = clearcoat;
	material.clearcoatRoughness = clearcoatRoughness;
	material.clearcoatF0 = vec3( 0.04 );
	material.clearcoatF90 = 1.0;
	#ifdef USE_CLEARCOATMAP
		material.clearcoat *= texture2D( clearcoatMap, vClearcoatMapUv ).x;
	#endif
	#ifdef USE_CLEARCOAT_ROUGHNESSMAP
		material.clearcoatRoughness *= texture2D( clearcoatRoughnessMap, vClearcoatRoughnessMapUv ).y;
	#endif
	material.clearcoat = saturate( material.clearcoat );	material.clearcoatRoughness = max( material.clearcoatRoughness, 0.0525 );
	material.clearcoatRoughness += geometryRoughness;
	material.clearcoatRoughness = min( material.clearcoatRoughness, 1.0 );
#endif
#ifdef USE_DISPERSION
	material.dispersion = dispersion;
#endif
#ifdef USE_IRIDESCENCE
	material.iridescence = iridescence;
	material.iridescenceIOR = iridescenceIOR;
	#ifdef USE_IRIDESCENCEMAP
		material.iridescence *= texture2D( iridescenceMap, vIridescenceMapUv ).r;
	#endif
	#ifdef USE_IRIDESCENCE_THICKNESSMAP
		material.iridescenceThickness = (iridescenceThicknessMaximum - iridescenceThicknessMinimum) * texture2D( iridescenceThicknessMap, vIridescenceThicknessMapUv ).g + iridescenceThicknessMinimum;
	#else
		material.iridescenceThickness = iridescenceThicknessMaximum;
	#endif
#endif
#ifdef USE_SHEEN
	material.sheenColor = sheenColor;
	#ifdef USE_SHEEN_COLORMAP
		material.sheenColor *= texture2D( sheenColorMap, vSheenColorMapUv ).rgb;
	#endif
	material.sheenRoughness = clamp( sheenRoughness, 0.0001, 1.0 );
	#ifdef USE_SHEEN_ROUGHNESSMAP
		material.sheenRoughness *= texture2D( sheenRoughnessMap, vSheenRoughnessMapUv ).a;
	#endif
#endif
#ifdef USE_ANISOTROPY
	#ifdef USE_ANISOTROPYMAP
		mat2 anisotropyMat = mat2( anisotropyVector.x, anisotropyVector.y, - anisotropyVector.y, anisotropyVector.x );
		vec3 anisotropyPolar = texture2D( anisotropyMap, vAnisotropyMapUv ).rgb;
		vec2 anisotropyV = anisotropyMat * normalize( 2.0 * anisotropyPolar.rg - vec2( 1.0 ) ) * anisotropyPolar.b;
	#else
		vec2 anisotropyV = anisotropyVector;
	#endif
	material.anisotropy = length( anisotropyV );
	if( material.anisotropy == 0.0 ) {
		anisotropyV = vec2( 1.0, 0.0 );
	} else {
		anisotropyV /= material.anisotropy;
		material.anisotropy = saturate( material.anisotropy );
	}
	material.alphaT = mix( pow2( material.roughness ), 1.0, pow2( material.anisotropy ) );
	material.anisotropyT = tbn[ 0 ] * anisotropyV.x + tbn[ 1 ] * anisotropyV.y;
	material.anisotropyB = tbn[ 1 ] * anisotropyV.x - tbn[ 0 ] * anisotropyV.y;
#endif`,gx=`uniform sampler2D dfgLUT;
struct PhysicalMaterial {
	vec3 diffuseColor;
	vec3 diffuseContribution;
	vec3 specularColor;
	vec3 specularColorBlended;
	float roughness;
	float metalness;
	float specularF90;
	float dispersion;
	#ifdef USE_CLEARCOAT
		float clearcoat;
		float clearcoatRoughness;
		vec3 clearcoatF0;
		float clearcoatF90;
	#endif
	#ifdef USE_IRIDESCENCE
		float iridescence;
		float iridescenceIOR;
		float iridescenceThickness;
		vec3 iridescenceFresnel;
		vec3 iridescenceF0;
		vec3 iridescenceFresnelDielectric;
		vec3 iridescenceFresnelMetallic;
	#endif
	#ifdef USE_SHEEN
		vec3 sheenColor;
		float sheenRoughness;
	#endif
	#ifdef IOR
		float ior;
	#endif
	#ifdef USE_TRANSMISSION
		float transmission;
		float transmissionAlpha;
		float thickness;
		float attenuationDistance;
		vec3 attenuationColor;
	#endif
	#ifdef USE_ANISOTROPY
		float anisotropy;
		float alphaT;
		vec3 anisotropyT;
		vec3 anisotropyB;
	#endif
};
vec3 clearcoatSpecularDirect = vec3( 0.0 );
vec3 clearcoatSpecularIndirect = vec3( 0.0 );
vec3 sheenSpecularDirect = vec3( 0.0 );
vec3 sheenSpecularIndirect = vec3(0.0 );
vec3 Schlick_to_F0( const in vec3 f, const in float f90, const in float dotVH ) {
    float x = clamp( 1.0 - dotVH, 0.0, 1.0 );
    float x2 = x * x;
    float x5 = clamp( x * x2 * x2, 0.0, 0.9999 );
    return ( f - vec3( f90 ) * x5 ) / ( 1.0 - x5 );
}
float V_GGX_SmithCorrelated( const in float alpha, const in float dotNL, const in float dotNV ) {
	float a2 = pow2( alpha );
	float gv = dotNL * sqrt( a2 + ( 1.0 - a2 ) * pow2( dotNV ) );
	float gl = dotNV * sqrt( a2 + ( 1.0 - a2 ) * pow2( dotNL ) );
	return 0.5 / max( gv + gl, EPSILON );
}
float D_GGX( const in float alpha, const in float dotNH ) {
	float a2 = pow2( alpha );
	float denom = pow2( dotNH ) * ( a2 - 1.0 ) + 1.0;
	return RECIPROCAL_PI * a2 / pow2( denom );
}
#ifdef USE_ANISOTROPY
	float V_GGX_SmithCorrelated_Anisotropic( const in float alphaT, const in float alphaB, const in float dotTV, const in float dotBV, const in float dotTL, const in float dotBL, const in float dotNV, const in float dotNL ) {
		float gv = dotNL * length( vec3( alphaT * dotTV, alphaB * dotBV, dotNV ) );
		float gl = dotNV * length( vec3( alphaT * dotTL, alphaB * dotBL, dotNL ) );
		return 0.5 / max( gv + gl, EPSILON );
	}
	float D_GGX_Anisotropic( const in float alphaT, const in float alphaB, const in float dotNH, const in float dotTH, const in float dotBH ) {
		float a2 = alphaT * alphaB;
		highp vec3 v = vec3( alphaB * dotTH, alphaT * dotBH, a2 * dotNH );
		highp float v2 = dot( v, v );
		float w2 = a2 / v2;
		return RECIPROCAL_PI * a2 * pow2 ( w2 );
	}
#endif
#ifdef USE_CLEARCOAT
	vec3 BRDF_GGX_Clearcoat( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material) {
		vec3 f0 = material.clearcoatF0;
		float f90 = material.clearcoatF90;
		float roughness = material.clearcoatRoughness;
		float alpha = pow2( roughness );
		vec3 halfDir = normalize( lightDir + viewDir );
		float dotNL = saturate( dot( normal, lightDir ) );
		float dotNV = saturate( dot( normal, viewDir ) );
		float dotNH = saturate( dot( normal, halfDir ) );
		float dotVH = saturate( dot( viewDir, halfDir ) );
		vec3 F = F_Schlick( f0, f90, dotVH );
		float V = V_GGX_SmithCorrelated( alpha, dotNL, dotNV );
		float D = D_GGX( alpha, dotNH );
		return F * ( V * D );
	}
#endif
vec3 BRDF_GGX( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material ) {
	vec3 f0 = material.specularColorBlended;
	float f90 = material.specularF90;
	float roughness = material.roughness;
	float alpha = pow2( roughness );
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	float dotNH = saturate( dot( normal, halfDir ) );
	float dotVH = saturate( dot( viewDir, halfDir ) );
	vec3 F = F_Schlick( f0, f90, dotVH );
	#ifdef USE_IRIDESCENCE
		F = mix( F, material.iridescenceFresnel, material.iridescence );
	#endif
	#ifdef USE_ANISOTROPY
		float dotTL = dot( material.anisotropyT, lightDir );
		float dotTV = dot( material.anisotropyT, viewDir );
		float dotTH = dot( material.anisotropyT, halfDir );
		float dotBL = dot( material.anisotropyB, lightDir );
		float dotBV = dot( material.anisotropyB, viewDir );
		float dotBH = dot( material.anisotropyB, halfDir );
		float V = V_GGX_SmithCorrelated_Anisotropic( material.alphaT, alpha, dotTV, dotBV, dotTL, dotBL, dotNV, dotNL );
		float D = D_GGX_Anisotropic( material.alphaT, alpha, dotNH, dotTH, dotBH );
	#else
		float V = V_GGX_SmithCorrelated( alpha, dotNL, dotNV );
		float D = D_GGX( alpha, dotNH );
	#endif
	return F * ( V * D );
}
vec2 LTC_Uv( const in vec3 N, const in vec3 V, const in float roughness ) {
	const float LUT_SIZE = 64.0;
	const float LUT_SCALE = ( LUT_SIZE - 1.0 ) / LUT_SIZE;
	const float LUT_BIAS = 0.5 / LUT_SIZE;
	float dotNV = saturate( dot( N, V ) );
	vec2 uv = vec2( roughness, sqrt( 1.0 - dotNV ) );
	uv = uv * LUT_SCALE + LUT_BIAS;
	return uv;
}
float LTC_ClippedSphereFormFactor( const in vec3 f ) {
	float l = length( f );
	return max( ( l * l + f.z ) / ( l + 1.0 ), 0.0 );
}
vec3 LTC_EdgeVectorFormFactor( const in vec3 v1, const in vec3 v2 ) {
	float x = dot( v1, v2 );
	float y = abs( x );
	float a = 0.8543985 + ( 0.4965155 + 0.0145206 * y ) * y;
	float b = 3.4175940 + ( 4.1616724 + y ) * y;
	float v = a / b;
	float theta_sintheta = ( x > 0.0 ) ? v : 0.5 * inversesqrt( max( 1.0 - x * x, 1e-7 ) ) - v;
	return cross( v1, v2 ) * theta_sintheta;
}
vec3 LTC_Evaluate( const in vec3 N, const in vec3 V, const in vec3 P, const in mat3 mInv, const in vec3 rectCoords[ 4 ] ) {
	vec3 v1 = rectCoords[ 1 ] - rectCoords[ 0 ];
	vec3 v2 = rectCoords[ 3 ] - rectCoords[ 0 ];
	vec3 lightNormal = cross( v1, v2 );
	if( dot( lightNormal, P - rectCoords[ 0 ] ) < 0.0 ) return vec3( 0.0 );
	vec3 T1, T2;
	T1 = normalize( V - N * dot( V, N ) );
	T2 = - cross( N, T1 );
	mat3 mat = mInv * transpose( mat3( T1, T2, N ) );
	vec3 coords[ 4 ];
	coords[ 0 ] = mat * ( rectCoords[ 0 ] - P );
	coords[ 1 ] = mat * ( rectCoords[ 1 ] - P );
	coords[ 2 ] = mat * ( rectCoords[ 2 ] - P );
	coords[ 3 ] = mat * ( rectCoords[ 3 ] - P );
	coords[ 0 ] = normalize( coords[ 0 ] );
	coords[ 1 ] = normalize( coords[ 1 ] );
	coords[ 2 ] = normalize( coords[ 2 ] );
	coords[ 3 ] = normalize( coords[ 3 ] );
	vec3 vectorFormFactor = vec3( 0.0 );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 0 ], coords[ 1 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 1 ], coords[ 2 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 2 ], coords[ 3 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 3 ], coords[ 0 ] );
	float result = LTC_ClippedSphereFormFactor( vectorFormFactor );
	return vec3( result );
}
#if defined( USE_SHEEN )
float D_Charlie( float roughness, float dotNH ) {
	float alpha = pow2( roughness );
	float invAlpha = 1.0 / alpha;
	float cos2h = dotNH * dotNH;
	float sin2h = max( 1.0 - cos2h, 0.0078125 );
	return ( 2.0 + invAlpha ) * pow( sin2h, invAlpha * 0.5 ) / ( 2.0 * PI );
}
float V_Neubelt( float dotNV, float dotNL ) {
	return saturate( 1.0 / ( 4.0 * ( dotNL + dotNV - dotNL * dotNV ) ) );
}
vec3 BRDF_Sheen( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, vec3 sheenColor, const in float sheenRoughness ) {
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	float dotNH = saturate( dot( normal, halfDir ) );
	float D = D_Charlie( sheenRoughness, dotNH );
	float V = V_Neubelt( dotNV, dotNL );
	return sheenColor * ( D * V );
}
#endif
float IBLSheenBRDF( const in vec3 normal, const in vec3 viewDir, const in float roughness ) {
	float dotNV = saturate( dot( normal, viewDir ) );
	float r2 = roughness * roughness;
	float rInv = 1.0 / ( roughness + 0.1 );
	float a = -1.9362 + 1.0678 * roughness + 0.4573 * r2 - 0.8469 * rInv;
	float b = -0.6014 + 0.5538 * roughness - 0.4670 * r2 - 0.1255 * rInv;
	float DG = exp( a * dotNV + b );
	return saturate( DG );
}
vec3 EnvironmentBRDF( const in vec3 normal, const in vec3 viewDir, const in vec3 specularColor, const in float specularF90, const in float roughness ) {
	float dotNV = saturate( dot( normal, viewDir ) );
	vec2 fab = texture2D( dfgLUT, vec2( roughness, dotNV ) ).rg;
	return specularColor * fab.x + specularF90 * fab.y;
}
#ifdef USE_IRIDESCENCE
void computeMultiscatteringIridescence( const in vec3 normal, const in vec3 viewDir, const in vec3 specularColor, const in float specularF90, const in float iridescence, const in vec3 iridescenceF0, const in float roughness, inout vec3 singleScatter, inout vec3 multiScatter ) {
#else
void computeMultiscattering( const in vec3 normal, const in vec3 viewDir, const in vec3 specularColor, const in float specularF90, const in float roughness, inout vec3 singleScatter, inout vec3 multiScatter ) {
#endif
	float dotNV = saturate( dot( normal, viewDir ) );
	vec2 fab = texture2D( dfgLUT, vec2( roughness, dotNV ) ).rg;
	#ifdef USE_IRIDESCENCE
		vec3 Fr = mix( specularColor, iridescenceF0, iridescence );
	#else
		vec3 Fr = specularColor;
	#endif
	vec3 FssEss = Fr * fab.x + specularF90 * fab.y;
	float Ess = fab.x + fab.y;
	float Ems = 1.0 - Ess;
	vec3 Favg = Fr + ( 1.0 - Fr ) * 0.047619;	vec3 Fms = FssEss * Favg / ( 1.0 - Ems * Favg );
	singleScatter += FssEss;
	multiScatter += Fms * Ems;
}
vec3 BRDF_GGX_Multiscatter( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material ) {
	vec3 singleScatter = BRDF_GGX( lightDir, viewDir, normal, material );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	vec2 dfgV = texture2D( dfgLUT, vec2( material.roughness, dotNV ) ).rg;
	vec2 dfgL = texture2D( dfgLUT, vec2( material.roughness, dotNL ) ).rg;
	vec3 FssEss_V = material.specularColorBlended * dfgV.x + material.specularF90 * dfgV.y;
	vec3 FssEss_L = material.specularColorBlended * dfgL.x + material.specularF90 * dfgL.y;
	float Ess_V = dfgV.x + dfgV.y;
	float Ess_L = dfgL.x + dfgL.y;
	float Ems_V = 1.0 - Ess_V;
	float Ems_L = 1.0 - Ess_L;
	vec3 Favg = material.specularColorBlended + ( 1.0 - material.specularColorBlended ) * 0.047619;
	vec3 Fms = FssEss_V * FssEss_L * Favg / ( 1.0 - Ems_V * Ems_L * Favg + EPSILON );
	float compensationFactor = Ems_V * Ems_L;
	vec3 multiScatter = Fms * compensationFactor;
	return singleScatter + multiScatter;
}
#if NUM_RECT_AREA_LIGHTS > 0
	void RE_Direct_RectArea_Physical( const in RectAreaLight rectAreaLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
		vec3 normal = geometryNormal;
		vec3 viewDir = geometryViewDir;
		vec3 position = geometryPosition;
		vec3 lightPos = rectAreaLight.position;
		vec3 halfWidth = rectAreaLight.halfWidth;
		vec3 halfHeight = rectAreaLight.halfHeight;
		vec3 lightColor = rectAreaLight.color;
		float roughness = material.roughness;
		vec3 rectCoords[ 4 ];
		rectCoords[ 0 ] = lightPos + halfWidth - halfHeight;		rectCoords[ 1 ] = lightPos - halfWidth - halfHeight;
		rectCoords[ 2 ] = lightPos - halfWidth + halfHeight;
		rectCoords[ 3 ] = lightPos + halfWidth + halfHeight;
		vec2 uv = LTC_Uv( normal, viewDir, roughness );
		vec4 t1 = texture2D( ltc_1, uv );
		vec4 t2 = texture2D( ltc_2, uv );
		mat3 mInv = mat3(
			vec3( t1.x, 0, t1.y ),
			vec3(    0, 1,    0 ),
			vec3( t1.z, 0, t1.w )
		);
		vec3 fresnel = ( material.specularColorBlended * t2.x + ( material.specularF90 - material.specularColorBlended ) * t2.y );
		reflectedLight.directSpecular += lightColor * fresnel * LTC_Evaluate( normal, viewDir, position, mInv, rectCoords );
		reflectedLight.directDiffuse += lightColor * material.diffuseContribution * LTC_Evaluate( normal, viewDir, position, mat3( 1.0 ), rectCoords );
		#ifdef USE_CLEARCOAT
			vec3 Ncc = geometryClearcoatNormal;
			vec2 uvClearcoat = LTC_Uv( Ncc, viewDir, material.clearcoatRoughness );
			vec4 t1Clearcoat = texture2D( ltc_1, uvClearcoat );
			vec4 t2Clearcoat = texture2D( ltc_2, uvClearcoat );
			mat3 mInvClearcoat = mat3(
				vec3( t1Clearcoat.x, 0, t1Clearcoat.y ),
				vec3(             0, 1,             0 ),
				vec3( t1Clearcoat.z, 0, t1Clearcoat.w )
			);
			vec3 fresnelClearcoat = material.clearcoatF0 * t2Clearcoat.x + ( material.clearcoatF90 - material.clearcoatF0 ) * t2Clearcoat.y;
			clearcoatSpecularDirect += lightColor * fresnelClearcoat * LTC_Evaluate( Ncc, viewDir, position, mInvClearcoat, rectCoords );
		#endif
	}
#endif
void RE_Direct_Physical( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	#ifdef USE_CLEARCOAT
		float dotNLcc = saturate( dot( geometryClearcoatNormal, directLight.direction ) );
		vec3 ccIrradiance = dotNLcc * directLight.color;
		clearcoatSpecularDirect += ccIrradiance * BRDF_GGX_Clearcoat( directLight.direction, geometryViewDir, geometryClearcoatNormal, material );
	#endif
	#ifdef USE_SHEEN
 
 		sheenSpecularDirect += irradiance * BRDF_Sheen( directLight.direction, geometryViewDir, geometryNormal, material.sheenColor, material.sheenRoughness );
 
 		float sheenAlbedoV = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
 		float sheenAlbedoL = IBLSheenBRDF( geometryNormal, directLight.direction, material.sheenRoughness );
 
 		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * max( sheenAlbedoV, sheenAlbedoL );
 
 		irradiance *= sheenEnergyComp;
 
 	#endif
	reflectedLight.directSpecular += irradiance * BRDF_GGX_Multiscatter( directLight.direction, geometryViewDir, geometryNormal, material );
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseContribution );
}
void RE_IndirectDiffuse_Physical( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
	vec3 diffuse = irradiance * BRDF_Lambert( material.diffuseContribution );
	#ifdef USE_SHEEN
		float sheenAlbedo = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * sheenAlbedo;
		diffuse *= sheenEnergyComp;
	#endif
	reflectedLight.indirectDiffuse += diffuse;
}
void RE_IndirectSpecular_Physical( const in vec3 radiance, const in vec3 irradiance, const in vec3 clearcoatRadiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight) {
	#ifdef USE_CLEARCOAT
		clearcoatSpecularIndirect += clearcoatRadiance * EnvironmentBRDF( geometryClearcoatNormal, geometryViewDir, material.clearcoatF0, material.clearcoatF90, material.clearcoatRoughness );
	#endif
	#ifdef USE_SHEEN
		sheenSpecularIndirect += irradiance * material.sheenColor * IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness ) * RECIPROCAL_PI;
 	#endif
	vec3 singleScatteringDielectric = vec3( 0.0 );
	vec3 multiScatteringDielectric = vec3( 0.0 );
	vec3 singleScatteringMetallic = vec3( 0.0 );
	vec3 multiScatteringMetallic = vec3( 0.0 );
	#ifdef USE_IRIDESCENCE
		computeMultiscatteringIridescence( geometryNormal, geometryViewDir, material.specularColor, material.specularF90, material.iridescence, material.iridescenceFresnelDielectric, material.roughness, singleScatteringDielectric, multiScatteringDielectric );
		computeMultiscatteringIridescence( geometryNormal, geometryViewDir, material.diffuseColor, material.specularF90, material.iridescence, material.iridescenceFresnelMetallic, material.roughness, singleScatteringMetallic, multiScatteringMetallic );
	#else
		computeMultiscattering( geometryNormal, geometryViewDir, material.specularColor, material.specularF90, material.roughness, singleScatteringDielectric, multiScatteringDielectric );
		computeMultiscattering( geometryNormal, geometryViewDir, material.diffuseColor, material.specularF90, material.roughness, singleScatteringMetallic, multiScatteringMetallic );
	#endif
	vec3 singleScattering = mix( singleScatteringDielectric, singleScatteringMetallic, material.metalness );
	vec3 multiScattering = mix( multiScatteringDielectric, multiScatteringMetallic, material.metalness );
	vec3 totalScatteringDielectric = singleScatteringDielectric + multiScatteringDielectric;
	vec3 diffuse = material.diffuseContribution * ( 1.0 - totalScatteringDielectric );
	vec3 cosineWeightedIrradiance = irradiance * RECIPROCAL_PI;
	vec3 indirectSpecular = radiance * singleScattering;
	indirectSpecular += multiScattering * cosineWeightedIrradiance;
	vec3 indirectDiffuse = diffuse * cosineWeightedIrradiance;
	#ifdef USE_SHEEN
		float sheenAlbedo = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * sheenAlbedo;
		indirectSpecular *= sheenEnergyComp;
		indirectDiffuse *= sheenEnergyComp;
	#endif
	reflectedLight.indirectSpecular += indirectSpecular;
	reflectedLight.indirectDiffuse += indirectDiffuse;
}
#define RE_Direct				RE_Direct_Physical
#define RE_Direct_RectArea		RE_Direct_RectArea_Physical
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Physical
#define RE_IndirectSpecular		RE_IndirectSpecular_Physical
float computeSpecularOcclusion( const in float dotNV, const in float ambientOcclusion, const in float roughness ) {
	return saturate( pow( dotNV + ambientOcclusion, exp2( - 16.0 * roughness - 1.0 ) ) - 1.0 + ambientOcclusion );
}`,_x=`
vec3 geometryPosition = - vViewPosition;
vec3 geometryNormal = normal;
vec3 geometryViewDir = ( isOrthographic ) ? vec3( 0, 0, 1 ) : normalize( vViewPosition );
vec3 geometryClearcoatNormal = vec3( 0.0 );
#ifdef USE_CLEARCOAT
	geometryClearcoatNormal = clearcoatNormal;
#endif
#ifdef USE_IRIDESCENCE
	float dotNVi = saturate( dot( normal, geometryViewDir ) );
	if ( material.iridescenceThickness == 0.0 ) {
		material.iridescence = 0.0;
	} else {
		material.iridescence = saturate( material.iridescence );
	}
	if ( material.iridescence > 0.0 ) {
		material.iridescenceFresnelDielectric = evalIridescence( 1.0, material.iridescenceIOR, dotNVi, material.iridescenceThickness, material.specularColor );
		material.iridescenceFresnelMetallic = evalIridescence( 1.0, material.iridescenceIOR, dotNVi, material.iridescenceThickness, material.diffuseColor );
		material.iridescenceFresnel = mix( material.iridescenceFresnelDielectric, material.iridescenceFresnelMetallic, material.metalness );
		material.iridescenceF0 = Schlick_to_F0( material.iridescenceFresnel, 1.0, dotNVi );
	}
#endif
IncidentLight directLight;
#if ( NUM_POINT_LIGHTS > 0 ) && defined( RE_Direct )
	PointLight pointLight;
	#if defined( USE_SHADOWMAP ) && NUM_POINT_LIGHT_SHADOWS > 0
	PointLightShadow pointLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_POINT_LIGHTS; i ++ ) {
		pointLight = pointLights[ i ];
		getPointLightInfo( pointLight, geometryPosition, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_POINT_LIGHT_SHADOWS ) && ( defined( SHADOWMAP_TYPE_PCF ) || defined( SHADOWMAP_TYPE_BASIC ) )
		pointLightShadow = pointLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getPointShadow( pointShadowMap[ i ], pointLightShadow.shadowMapSize, pointLightShadow.shadowIntensity, pointLightShadow.shadowBias, pointLightShadow.shadowRadius, vPointShadowCoord[ i ], pointLightShadow.shadowCameraNear, pointLightShadow.shadowCameraFar ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_SPOT_LIGHTS > 0 ) && defined( RE_Direct )
	SpotLight spotLight;
	vec4 spotColor;
	vec3 spotLightCoord;
	bool inSpotLightMap;
	#if defined( USE_SHADOWMAP ) && NUM_SPOT_LIGHT_SHADOWS > 0
	SpotLightShadow spotLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHTS; i ++ ) {
		spotLight = spotLights[ i ];
		getSpotLightInfo( spotLight, geometryPosition, directLight );
		#if ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS )
		#define SPOT_LIGHT_MAP_INDEX UNROLLED_LOOP_INDEX
		#elif ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
		#define SPOT_LIGHT_MAP_INDEX NUM_SPOT_LIGHT_MAPS
		#else
		#define SPOT_LIGHT_MAP_INDEX ( UNROLLED_LOOP_INDEX - NUM_SPOT_LIGHT_SHADOWS + NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS )
		#endif
		#if ( SPOT_LIGHT_MAP_INDEX < NUM_SPOT_LIGHT_MAPS )
			spotLightCoord = vSpotLightCoord[ i ].xyz / vSpotLightCoord[ i ].w;
			inSpotLightMap = all( lessThan( abs( spotLightCoord * 2. - 1. ), vec3( 1.0 ) ) );
			spotColor = texture2D( spotLightMap[ SPOT_LIGHT_MAP_INDEX ], spotLightCoord.xy );
			directLight.color = inSpotLightMap ? directLight.color * spotColor.rgb : directLight.color;
		#endif
		#undef SPOT_LIGHT_MAP_INDEX
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
		spotLightShadow = spotLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getShadow( spotShadowMap[ i ], spotLightShadow.shadowMapSize, spotLightShadow.shadowIntensity, spotLightShadow.shadowBias, spotLightShadow.shadowRadius, vSpotLightCoord[ i ] ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_DIR_LIGHTS > 0 ) && defined( RE_Direct )
	DirectionalLight directionalLight;
	#if defined( USE_SHADOWMAP ) && NUM_DIR_LIGHT_SHADOWS > 0
	DirectionalLightShadow directionalLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_DIR_LIGHTS; i ++ ) {
		directionalLight = directionalLights[ i ];
		getDirectionalLightInfo( directionalLight, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_DIR_LIGHT_SHADOWS )
		directionalLightShadow = directionalLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getShadow( directionalShadowMap[ i ], directionalLightShadow.shadowMapSize, directionalLightShadow.shadowIntensity, directionalLightShadow.shadowBias, directionalLightShadow.shadowRadius, vDirectionalShadowCoord[ i ] ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_RECT_AREA_LIGHTS > 0 ) && defined( RE_Direct_RectArea )
	RectAreaLight rectAreaLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_RECT_AREA_LIGHTS; i ++ ) {
		rectAreaLight = rectAreaLights[ i ];
		RE_Direct_RectArea( rectAreaLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if defined( RE_IndirectDiffuse )
	vec3 iblIrradiance = vec3( 0.0 );
	vec3 irradiance = getAmbientLightIrradiance( ambientLightColor );
	#if defined( USE_LIGHT_PROBES )
		irradiance += getLightProbeIrradiance( lightProbe, geometryNormal );
	#endif
	#if ( NUM_HEMI_LIGHTS > 0 )
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_HEMI_LIGHTS; i ++ ) {
			irradiance += getHemisphereLightIrradiance( hemisphereLights[ i ], geometryNormal );
		}
		#pragma unroll_loop_end
	#endif
	#ifdef USE_LIGHT_PROBES_GRID
		vec3 probeWorldPos = ( ( vec4( geometryPosition, 1.0 ) - viewMatrix[ 3 ] ) * viewMatrix ).xyz;
		vec3 probeWorldNormal = transformNormalByInverseViewMatrix( geometryNormal, viewMatrix );
		irradiance += getLightProbeGridIrradiance( probeWorldPos, probeWorldNormal );
	#endif
#endif
#if defined( RE_IndirectSpecular )
	vec3 radiance = vec3( 0.0 );
	vec3 clearcoatRadiance = vec3( 0.0 );
#endif`,xx=`#if defined( RE_IndirectDiffuse )
	#ifdef USE_LIGHTMAP
		vec4 lightMapTexel = texture2D( lightMap, vLightMapUv );
		vec3 lightMapIrradiance = lightMapTexel.rgb * lightMapIntensity;
		irradiance += lightMapIrradiance;
	#endif
	#if defined( USE_ENVMAP ) && defined( ENVMAP_TYPE_CUBE_UV )
		#if defined( STANDARD ) || defined( LAMBERT ) || defined( PHONG )
			iblIrradiance += getIBLIrradiance( geometryNormal );
		#endif
	#endif
#endif
#if defined( USE_ENVMAP ) && defined( RE_IndirectSpecular )
	#ifdef USE_ANISOTROPY
		radiance += getIBLAnisotropyRadiance( geometryViewDir, geometryNormal, material.roughness, material.anisotropyB, material.anisotropy );
	#else
		radiance += getIBLRadiance( geometryViewDir, geometryNormal, material.roughness );
	#endif
	#ifdef USE_CLEARCOAT
		clearcoatRadiance += getIBLRadiance( geometryViewDir, geometryClearcoatNormal, material.clearcoatRoughness );
	#endif
#endif`,vx=`#if defined( RE_IndirectDiffuse )
	#if defined( LAMBERT ) || defined( PHONG )
		irradiance += iblIrradiance;
	#endif
	RE_IndirectDiffuse( irradiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif
#if defined( RE_IndirectSpecular )
	RE_IndirectSpecular( radiance, iblIrradiance, clearcoatRadiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif`,Sx=`#ifdef USE_LIGHT_PROBES_GRID
uniform highp sampler3D probesSH;
uniform vec3 probesMin;
uniform vec3 probesMax;
uniform vec3 probesResolution;
vec3 getLightProbeGridIrradiance( vec3 worldPos, vec3 worldNormal ) {
	vec3 res = probesResolution;
	vec3 gridRange = probesMax - probesMin;
	vec3 resMinusOne = res - 1.0;
	vec3 probeSpacing = gridRange / resMinusOne;
	vec3 samplePos = worldPos + worldNormal * probeSpacing * 0.5;
	vec3 uvw = clamp( ( samplePos - probesMin ) / gridRange, 0.0, 1.0 );
	uvw = uvw * resMinusOne / res + 0.5 / res;
	float nz          = res.z;
	float paddedSlices = nz + 2.0;
	float atlasDepth  = 7.0 * paddedSlices;
	float uvZBase     = uvw.z * nz + 1.0;
	vec4 s0 = texture( probesSH, vec3( uvw.xy, ( uvZBase                       ) / atlasDepth ) );
	vec4 s1 = texture( probesSH, vec3( uvw.xy, ( uvZBase +       paddedSlices   ) / atlasDepth ) );
	vec4 s2 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 2.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s3 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 3.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s4 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 4.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s5 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 5.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s6 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 6.0 * paddedSlices   ) / atlasDepth ) );
	vec3 c0 = s0.xyz;
	vec3 c1 = vec3( s0.w, s1.xy );
	vec3 c2 = vec3( s1.zw, s2.x );
	vec3 c3 = s2.yzw;
	vec3 c4 = s3.xyz;
	vec3 c5 = vec3( s3.w, s4.xy );
	vec3 c6 = vec3( s4.zw, s5.x );
	vec3 c7 = s5.yzw;
	vec3 c8 = s6.xyz;
	float x = worldNormal.x, y = worldNormal.y, z = worldNormal.z;
	vec3 result = c0 * 0.886227;
	result += c1 * 2.0 * 0.511664 * y;
	result += c2 * 2.0 * 0.511664 * z;
	result += c3 * 2.0 * 0.511664 * x;
	result += c4 * 2.0 * 0.429043 * x * y;
	result += c5 * 2.0 * 0.429043 * y * z;
	result += c6 * ( 0.743125 * z * z - 0.247708 );
	result += c7 * 2.0 * 0.429043 * x * z;
	result += c8 * 0.429043 * ( x * x - y * y );
	return max( result, vec3( 0.0 ) );
}
#endif`,Mx=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	gl_FragDepth = vIsPerspective == 0.0 ? gl_FragCoord.z : log2( vFragDepth ) * logDepthBufFC * 0.5;
#endif`,yx=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	uniform float logDepthBufFC;
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,Ex=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,bx=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	vFragDepth = 1.0 + gl_Position.w;
	vIsPerspective = float( isPerspectiveMatrix( projectionMatrix ) );
#endif`,Tx=`#ifdef USE_MAP
	vec4 sampledDiffuseColor = texture2D( map, vMapUv );
	#ifdef DECODE_VIDEO_TEXTURE
		sampledDiffuseColor = sRGBTransferEOTF( sampledDiffuseColor );
	#endif
	diffuseColor *= sampledDiffuseColor;
#endif`,Ax=`#ifdef USE_MAP
	uniform sampler2D map;
#endif`,wx=`#if defined( USE_MAP ) || defined( USE_ALPHAMAP )
	#if defined( USE_POINTS_UV )
		vec2 uv = vUv;
	#else
		vec2 uv = ( uvTransform * vec3( gl_PointCoord.x, 1.0 - gl_PointCoord.y, 1 ) ).xy;
	#endif
#endif
#ifdef USE_MAP
	diffuseColor *= texture2D( map, uv );
#endif
#ifdef USE_ALPHAMAP
	diffuseColor.a *= texture2D( alphaMap, uv ).g;
#endif`,Rx=`#if defined( USE_POINTS_UV )
	varying vec2 vUv;
#else
	#if defined( USE_MAP ) || defined( USE_ALPHAMAP )
		uniform mat3 uvTransform;
	#endif
#endif
#ifdef USE_MAP
	uniform sampler2D map;
#endif
#ifdef USE_ALPHAMAP
	uniform sampler2D alphaMap;
#endif`,Cx=`float metalnessFactor = metalness;
#ifdef USE_METALNESSMAP
	vec4 texelMetalness = texture2D( metalnessMap, vMetalnessMapUv );
	metalnessFactor *= texelMetalness.b;
#endif`,Px=`#ifdef USE_METALNESSMAP
	uniform sampler2D metalnessMap;
#endif`,Dx=`#ifdef USE_INSTANCING_MORPH
	float morphTargetInfluences[ MORPHTARGETS_COUNT ];
	float morphTargetBaseInfluence = texelFetch( morphTexture, ivec2( 0, gl_InstanceID ), 0 ).r;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		morphTargetInfluences[i] =  texelFetch( morphTexture, ivec2( i + 1, gl_InstanceID ), 0 ).r;
	}
#endif`,Lx=`#if defined( USE_MORPHCOLORS )
	vColor *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		#if defined( USE_COLOR_ALPHA )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ) * morphTargetInfluences[ i ];
		#elif defined( USE_COLOR )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ).rgb * morphTargetInfluences[ i ];
		#endif
	}
#endif`,Ix=`#ifdef USE_MORPHNORMALS
	objectNormal *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) objectNormal += getMorph( gl_VertexID, i, 1 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,Ux=`#ifdef USE_MORPHTARGETS
	#ifndef USE_INSTANCING_MORPH
		uniform float morphTargetBaseInfluence;
		uniform float morphTargetInfluences[ MORPHTARGETS_COUNT ];
	#endif
	uniform sampler2DArray morphTargetsTexture;
	uniform ivec2 morphTargetsTextureSize;
	vec4 getMorph( const in int vertexIndex, const in int morphTargetIndex, const in int offset ) {
		int texelIndex = vertexIndex * MORPHTARGETS_TEXTURE_STRIDE + offset;
		int y = texelIndex / morphTargetsTextureSize.x;
		int x = texelIndex - y * morphTargetsTextureSize.x;
		ivec3 morphUV = ivec3( x, y, morphTargetIndex );
		return texelFetch( morphTargetsTexture, morphUV, 0 );
	}
#endif`,Nx=`#ifdef USE_MORPHTARGETS
	transformed *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) transformed += getMorph( gl_VertexID, i, 0 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,Fx=`float faceDirection = gl_FrontFacing ? 1.0 : - 1.0;
#ifdef FLAT_SHADED
	vec3 fdx = dFdx( vViewPosition );
	vec3 fdy = dFdy( vViewPosition );
	vec3 normal = normalize( cross( fdx, fdy ) );
#else
	vec3 normal = normalize( vNormal );
	#ifdef DOUBLE_SIDED
		normal *= faceDirection;
	#endif
#endif
#if defined( USE_NORMALMAP_TANGENTSPACE ) || defined( USE_CLEARCOAT_NORMALMAP ) || defined( USE_ANISOTROPY )
	#ifdef USE_TANGENT
		mat3 tbn = mat3( normalize( vTangent ), normalize( vBitangent ), normal );
	#else
		mat3 tbn = getTangentFrame( - vViewPosition, normal,
		#if defined( USE_NORMALMAP )
			vNormalMapUv
		#elif defined( USE_CLEARCOAT_NORMALMAP )
			vClearcoatNormalMapUv
		#else
			vUv
		#endif
		);
	#endif
	#ifdef DOUBLE_SIDED
		tbn[0] *= faceDirection;
		tbn[1] *= faceDirection;
	#endif
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	#ifdef USE_TANGENT
		mat3 tbn2 = mat3( normalize( vTangent ), normalize( vBitangent ), normal );
	#else
		mat3 tbn2 = getTangentFrame( - vViewPosition, normal, vClearcoatNormalMapUv );
	#endif
	#ifdef DOUBLE_SIDED
		tbn2[0] *= faceDirection;
		tbn2[1] *= faceDirection;
	#endif
#endif
vec3 nonPerturbedNormal = normal;`,Ox=`#ifdef USE_NORMALMAP_OBJECTSPACE
	normal = texture2D( normalMap, vNormalMapUv ).xyz * 2.0 - 1.0;
	#ifdef FLIP_SIDED
		normal = - normal;
	#endif
	#ifdef DOUBLE_SIDED
		normal = normal * faceDirection;
	#endif
	normal = normalize( normalMatrix * normal );
#elif defined( USE_NORMALMAP_TANGENTSPACE )
	vec3 mapN = texture2D( normalMap, vNormalMapUv ).xyz * 2.0 - 1.0;
	#if defined( USE_PACKED_NORMALMAP )
		mapN = vec3( mapN.xy, sqrt( saturate( 1.0 - dot( mapN.xy, mapN.xy ) ) ) );
	#endif
	mapN.xy *= normalScale;
	normal = normalize( tbn * mapN );
#elif defined( USE_BUMPMAP )
	normal = perturbNormalArb( - vViewPosition, normal, dHdxy_fwd(), faceDirection );
#endif`,Bx=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,zx=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,kx=`#ifndef FLAT_SHADED
	vNormal = normalize( transformedNormal );
	#ifdef USE_TANGENT
		vTangent = normalize( transformedTangent );
		vBitangent = normalize( cross( vNormal, vTangent ) * tangent.w );
		#ifdef FLIP_SIDED
			vBitangent = - vBitangent;
		#endif
	#endif
#endif`,Vx=`#ifdef USE_NORMALMAP
	uniform sampler2D normalMap;
	uniform vec2 normalScale;
#endif
#ifdef USE_NORMALMAP_OBJECTSPACE
	uniform mat3 normalMatrix;
#endif
#if ! defined ( USE_TANGENT ) && ( defined ( USE_NORMALMAP_TANGENTSPACE ) || defined ( USE_CLEARCOAT_NORMALMAP ) || defined( USE_ANISOTROPY ) )
	mat3 getTangentFrame( vec3 eye_pos, vec3 surf_norm, vec2 uv ) {
		vec3 q0 = dFdx( eye_pos.xyz );
		vec3 q1 = dFdy( eye_pos.xyz );
		vec2 st0 = dFdx( uv.st );
		vec2 st1 = dFdy( uv.st );
		vec3 N = surf_norm;
		vec3 q1perp = cross( q1, N );
		vec3 q0perp = cross( N, q0 );
		vec3 T = q1perp * st0.x + q0perp * st1.x;
		vec3 B = q1perp * st0.y + q0perp * st1.y;
		float det = max( dot( T, T ), dot( B, B ) );
		float scale = ( det == 0.0 ) ? 0.0 : inversesqrt( det );
		return mat3( T * scale, B * scale, N );
	}
#endif`,Gx=`#ifdef USE_CLEARCOAT
	vec3 clearcoatNormal = nonPerturbedNormal;
#endif`,Hx=`#ifdef USE_CLEARCOAT_NORMALMAP
	vec3 clearcoatMapN = texture2D( clearcoatNormalMap, vClearcoatNormalMapUv ).xyz * 2.0 - 1.0;
	clearcoatMapN.xy *= clearcoatNormalScale;
	clearcoatNormal = normalize( tbn2 * clearcoatMapN );
#endif`,Wx=`#ifdef USE_CLEARCOATMAP
	uniform sampler2D clearcoatMap;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	uniform sampler2D clearcoatNormalMap;
	uniform vec2 clearcoatNormalScale;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	uniform sampler2D clearcoatRoughnessMap;
#endif`,Xx=`#ifdef USE_IRIDESCENCEMAP
	uniform sampler2D iridescenceMap;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	uniform sampler2D iridescenceThicknessMap;
#endif`,Yx=`#ifdef OPAQUE
diffuseColor.a = 1.0;
#endif
#ifdef USE_TRANSMISSION
diffuseColor.a *= material.transmissionAlpha;
#endif
gl_FragColor = vec4( outgoingLight, diffuseColor.a );`,qx=`vec3 packNormalToRGB( const in vec3 normal ) {
	return normalize( normal ) * 0.5 + 0.5;
}
vec3 unpackRGBToNormal( const in vec3 rgb ) {
	return 2.0 * rgb.xyz - 1.0;
}
const float PackUpscale = 256. / 255.;const float UnpackDownscale = 255. / 256.;const float ShiftRight8 = 1. / 256.;
const float Inv255 = 1. / 255.;
const vec4 PackFactors = vec4( 1.0, 256.0, 256.0 * 256.0, 256.0 * 256.0 * 256.0 );
const vec2 UnpackFactors2 = vec2( UnpackDownscale, 1.0 / PackFactors.g );
const vec3 UnpackFactors3 = vec3( UnpackDownscale / PackFactors.rg, 1.0 / PackFactors.b );
const vec4 UnpackFactors4 = vec4( UnpackDownscale / PackFactors.rgb, 1.0 / PackFactors.a );
vec4 packDepthToRGBA( const in float v ) {
	if( v <= 0.0 )
		return vec4( 0., 0., 0., 0. );
	if( v >= 1.0 )
		return vec4( 1., 1., 1., 1. );
	float vuf;
	float af = modf( v * PackFactors.a, vuf );
	float bf = modf( vuf * ShiftRight8, vuf );
	float gf = modf( vuf * ShiftRight8, vuf );
	return vec4( vuf * Inv255, gf * PackUpscale, bf * PackUpscale, af );
}
vec3 packDepthToRGB( const in float v ) {
	if( v <= 0.0 )
		return vec3( 0., 0., 0. );
	if( v >= 1.0 )
		return vec3( 1., 1., 1. );
	float vuf;
	float bf = modf( v * PackFactors.b, vuf );
	float gf = modf( vuf * ShiftRight8, vuf );
	return vec3( vuf * Inv255, gf * PackUpscale, bf );
}
vec2 packDepthToRG( const in float v ) {
	if( v <= 0.0 )
		return vec2( 0., 0. );
	if( v >= 1.0 )
		return vec2( 1., 1. );
	float vuf;
	float gf = modf( v * 256., vuf );
	return vec2( vuf * Inv255, gf );
}
float unpackRGBAToDepth( const in vec4 v ) {
	return dot( v, UnpackFactors4 );
}
float unpackRGBToDepth( const in vec3 v ) {
	return dot( v, UnpackFactors3 );
}
float unpackRGToDepth( const in vec2 v ) {
	return v.r * UnpackFactors2.r + v.g * UnpackFactors2.g;
}
vec4 pack2HalfToRGBA( const in vec2 v ) {
	vec4 r = vec4( v.x, fract( v.x * 255.0 ), v.y, fract( v.y * 255.0 ) );
	return vec4( r.x - r.y / 255.0, r.y, r.z - r.w / 255.0, r.w );
}
vec2 unpackRGBATo2Half( const in vec4 v ) {
	return vec2( v.x + ( v.y / 255.0 ), v.z + ( v.w / 255.0 ) );
}
float viewZToOrthographicDepth( const in float viewZ, const in float near, const in float far ) {
	return ( viewZ + near ) / ( near - far );
}
float orthographicDepthToViewZ( const in float depth, const in float near, const in float far ) {
	#ifdef USE_REVERSED_DEPTH_BUFFER
	
		return depth * ( far - near ) - far;
	#else
		return depth * ( near - far ) - near;
	#endif
}
float viewZToPerspectiveDepth( const in float viewZ, const in float near, const in float far ) {
	return ( ( near + viewZ ) * far ) / ( ( far - near ) * viewZ );
}
float perspectiveDepthToViewZ( const in float depth, const in float near, const in float far ) {
	
	#ifdef USE_REVERSED_DEPTH_BUFFER
		return ( near * far ) / ( ( near - far ) * depth - near );
	#else
		return ( near * far ) / ( ( far - near ) * depth - far );
	#endif
}`,Kx=`#ifdef PREMULTIPLIED_ALPHA
	gl_FragColor.rgb *= gl_FragColor.a;
#endif`,$x=`vec4 mvPosition = vec4( transformed, 1.0 );
#ifdef USE_BATCHING
	mvPosition = batchingMatrix * mvPosition;
#endif
#ifdef USE_INSTANCING
	mvPosition = instanceMatrix * mvPosition;
#endif
mvPosition = modelViewMatrix * mvPosition;
gl_Position = projectionMatrix * mvPosition;`,Zx=`#ifdef DITHERING
	gl_FragColor.rgb = dithering( gl_FragColor.rgb );
#endif`,Jx=`#ifdef DITHERING
	vec3 dithering( vec3 color ) {
		float grid_position = rand( gl_FragCoord.xy );
		vec3 dither_shift_RGB = vec3( 0.25 / 255.0, -0.25 / 255.0, 0.25 / 255.0 );
		dither_shift_RGB = mix( 2.0 * dither_shift_RGB, -2.0 * dither_shift_RGB, grid_position );
		return color + dither_shift_RGB;
	}
#endif`,jx=`float roughnessFactor = roughness;
#ifdef USE_ROUGHNESSMAP
	vec4 texelRoughness = texture2D( roughnessMap, vRoughnessMapUv );
	roughnessFactor *= texelRoughness.g;
#endif`,Qx=`#ifdef USE_ROUGHNESSMAP
	uniform sampler2D roughnessMap;
#endif`,ev=`#if NUM_SPOT_LIGHT_COORDS > 0
	varying vec4 vSpotLightCoord[ NUM_SPOT_LIGHT_COORDS ];
#endif
#if NUM_SPOT_LIGHT_MAPS > 0
	uniform sampler2D spotLightMap[ NUM_SPOT_LIGHT_MAPS ];
#endif
#ifdef USE_SHADOWMAP
	#if NUM_DIR_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow directionalShadowMap[ NUM_DIR_LIGHT_SHADOWS ];
		#else
			uniform sampler2D directionalShadowMap[ NUM_DIR_LIGHT_SHADOWS ];
		#endif
		varying vec4 vDirectionalShadowCoord[ NUM_DIR_LIGHT_SHADOWS ];
		struct DirectionalLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform DirectionalLightShadow directionalLightShadows[ NUM_DIR_LIGHT_SHADOWS ];
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow spotShadowMap[ NUM_SPOT_LIGHT_SHADOWS ];
		#else
			uniform sampler2D spotShadowMap[ NUM_SPOT_LIGHT_SHADOWS ];
		#endif
		struct SpotLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SpotLightShadow spotLightShadows[ NUM_SPOT_LIGHT_SHADOWS ];
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform samplerCubeShadow pointShadowMap[ NUM_POINT_LIGHT_SHADOWS ];
		#elif defined( SHADOWMAP_TYPE_BASIC )
			uniform samplerCube pointShadowMap[ NUM_POINT_LIGHT_SHADOWS ];
		#endif
		varying vec4 vPointShadowCoord[ NUM_POINT_LIGHT_SHADOWS ];
		struct PointLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
			float shadowCameraNear;
			float shadowCameraFar;
		};
		uniform PointLightShadow pointLightShadows[ NUM_POINT_LIGHT_SHADOWS ];
	#endif
	#if defined( SHADOWMAP_TYPE_PCF )
		float interleavedGradientNoise( vec2 position ) {
			return fract( 52.9829189 * fract( dot( position, vec2( 0.06711056, 0.00583715 ) ) ) );
		}
		vec2 vogelDiskSample( int sampleIndex, int samplesCount, float phi ) {
			const float goldenAngle = 2.399963229728653;
			float r = sqrt( ( float( sampleIndex ) + 0.5 ) / float( samplesCount ) );
			float theta = float( sampleIndex ) * goldenAngle + phi;
			return vec2( cos( theta ), sin( theta ) ) * r;
		}
	#endif
	#if defined( SHADOWMAP_TYPE_PCF )
		float getShadow( sampler2DShadow shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			shadowCoord.z += shadowBias;
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				vec2 texelSize = vec2( 1.0 ) / shadowMapSize;
				float radius = shadowRadius * texelSize.x;
				float phi = interleavedGradientNoise( gl_FragCoord.xy ) * PI2;
				shadow = (
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 0, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 1, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 2, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 3, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 4, 5, phi ) * radius, shadowCoord.z ) )
				) * 0.2;
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#elif defined( SHADOWMAP_TYPE_VSM )
		float getShadow( sampler2D shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				shadowCoord.z -= shadowBias;
			#else
				shadowCoord.z += shadowBias;
			#endif
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				vec2 distribution = texture2D( shadowMap, shadowCoord.xy ).rg;
				float mean = distribution.x;
				float variance = distribution.y * distribution.y;
				#ifdef USE_REVERSED_DEPTH_BUFFER
					float hard_shadow = step( mean, shadowCoord.z );
				#else
					float hard_shadow = step( shadowCoord.z, mean );
				#endif
				
				if ( hard_shadow == 1.0 ) {
					shadow = 1.0;
				} else {
					variance = max( variance, 0.0000001 );
					float d = shadowCoord.z - mean;
					float p_max = variance / ( variance + d * d );
					p_max = clamp( ( p_max - 0.3 ) / 0.65, 0.0, 1.0 );
					shadow = max( hard_shadow, p_max );
				}
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#else
		float getShadow( sampler2D shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				shadowCoord.z -= shadowBias;
			#else
				shadowCoord.z += shadowBias;
			#endif
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				float depth = texture2D( shadowMap, shadowCoord.xy ).r;
				#ifdef USE_REVERSED_DEPTH_BUFFER
					shadow = step( depth, shadowCoord.z );
				#else
					shadow = step( shadowCoord.z, depth );
				#endif
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
	#if defined( SHADOWMAP_TYPE_PCF )
	float getPointShadow( samplerCubeShadow shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord, float shadowCameraNear, float shadowCameraFar ) {
		float shadow = 1.0;
		vec3 lightToPosition = shadowCoord.xyz;
		vec3 bd3D = normalize( lightToPosition );
		vec3 absVec = abs( lightToPosition );
		float viewSpaceZ = max( max( absVec.x, absVec.y ), absVec.z );
		if ( viewSpaceZ - shadowCameraFar <= 0.0 && viewSpaceZ - shadowCameraNear >= 0.0 ) {
			#ifdef USE_REVERSED_DEPTH_BUFFER
				float dp = ( shadowCameraNear * ( shadowCameraFar - viewSpaceZ ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
				dp -= shadowBias;
			#else
				float dp = ( shadowCameraFar * ( viewSpaceZ - shadowCameraNear ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
				dp += shadowBias;
			#endif
			float texelSize = shadowRadius / shadowMapSize.x;
			vec3 absDir = abs( bd3D );
			vec3 tangent = absDir.x > absDir.z ? vec3( 0.0, 1.0, 0.0 ) : vec3( 1.0, 0.0, 0.0 );
			tangent = normalize( cross( bd3D, tangent ) );
			vec3 bitangent = cross( bd3D, tangent );
			float phi = interleavedGradientNoise( gl_FragCoord.xy ) * PI2;
			vec2 sample0 = vogelDiskSample( 0, 5, phi );
			vec2 sample1 = vogelDiskSample( 1, 5, phi );
			vec2 sample2 = vogelDiskSample( 2, 5, phi );
			vec2 sample3 = vogelDiskSample( 3, 5, phi );
			vec2 sample4 = vogelDiskSample( 4, 5, phi );
			shadow = (
				texture( shadowMap, vec4( bd3D + ( tangent * sample0.x + bitangent * sample0.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample1.x + bitangent * sample1.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample2.x + bitangent * sample2.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample3.x + bitangent * sample3.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample4.x + bitangent * sample4.y ) * texelSize, dp ) )
			) * 0.2;
		}
		return mix( 1.0, shadow, shadowIntensity );
	}
	#elif defined( SHADOWMAP_TYPE_BASIC )
	float getPointShadow( samplerCube shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord, float shadowCameraNear, float shadowCameraFar ) {
		float shadow = 1.0;
		vec3 lightToPosition = shadowCoord.xyz;
		vec3 absVec = abs( lightToPosition );
		float viewSpaceZ = max( max( absVec.x, absVec.y ), absVec.z );
		if ( viewSpaceZ - shadowCameraFar <= 0.0 && viewSpaceZ - shadowCameraNear >= 0.0 ) {
			float dp = ( shadowCameraFar * ( viewSpaceZ - shadowCameraNear ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
			dp += shadowBias;
			vec3 bd3D = normalize( lightToPosition );
			float depth = textureCube( shadowMap, bd3D ).r;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				depth = 1.0 - depth;
			#endif
			shadow = step( dp, depth );
		}
		return mix( 1.0, shadow, shadowIntensity );
	}
	#endif
	#endif
#endif`,tv=`#if NUM_SPOT_LIGHT_COORDS > 0
	uniform mat4 spotLightMatrix[ NUM_SPOT_LIGHT_COORDS ];
	varying vec4 vSpotLightCoord[ NUM_SPOT_LIGHT_COORDS ];
#endif
#ifdef USE_SHADOWMAP
	#if NUM_DIR_LIGHT_SHADOWS > 0
		uniform mat4 directionalShadowMatrix[ NUM_DIR_LIGHT_SHADOWS ];
		varying vec4 vDirectionalShadowCoord[ NUM_DIR_LIGHT_SHADOWS ];
		struct DirectionalLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform DirectionalLightShadow directionalLightShadows[ NUM_DIR_LIGHT_SHADOWS ];
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
		struct SpotLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SpotLightShadow spotLightShadows[ NUM_SPOT_LIGHT_SHADOWS ];
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		uniform mat4 pointShadowMatrix[ NUM_POINT_LIGHT_SHADOWS ];
		varying vec4 vPointShadowCoord[ NUM_POINT_LIGHT_SHADOWS ];
		struct PointLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
			float shadowCameraNear;
			float shadowCameraFar;
		};
		uniform PointLightShadow pointLightShadows[ NUM_POINT_LIGHT_SHADOWS ];
	#endif
#endif`,nv=`#if ( defined( USE_SHADOWMAP ) && ( NUM_DIR_LIGHT_SHADOWS > 0 || NUM_POINT_LIGHT_SHADOWS > 0 ) ) || ( NUM_SPOT_LIGHT_COORDS > 0 )
	#ifdef HAS_NORMAL
		vec3 shadowWorldNormal = transformNormalByInverseViewMatrix( transformedNormal, viewMatrix );
	#else
		vec3 shadowWorldNormal = vec3( 0.0 );
	#endif
	vec4 shadowWorldPosition;
#endif
#if defined( USE_SHADOWMAP )
	#if NUM_DIR_LIGHT_SHADOWS > 0
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_DIR_LIGHT_SHADOWS; i ++ ) {
			shadowWorldPosition = worldPosition + vec4( shadowWorldNormal * directionalLightShadows[ i ].shadowNormalBias, 0 );
			vDirectionalShadowCoord[ i ] = directionalShadowMatrix[ i ] * shadowWorldPosition;
		}
		#pragma unroll_loop_end
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_POINT_LIGHT_SHADOWS; i ++ ) {
			shadowWorldPosition = worldPosition + vec4( shadowWorldNormal * pointLightShadows[ i ].shadowNormalBias, 0 );
			vPointShadowCoord[ i ] = pointShadowMatrix[ i ] * shadowWorldPosition;
		}
		#pragma unroll_loop_end
	#endif
#endif
#if NUM_SPOT_LIGHT_COORDS > 0
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHT_COORDS; i ++ ) {
		shadowWorldPosition = worldPosition;
		#if ( defined( USE_SHADOWMAP ) && UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
			shadowWorldPosition.xyz += shadowWorldNormal * spotLightShadows[ i ].shadowNormalBias;
		#endif
		vSpotLightCoord[ i ] = spotLightMatrix[ i ] * shadowWorldPosition;
	}
	#pragma unroll_loop_end
#endif`,iv=`float getShadowMask() {
	float shadow = 1.0;
	#ifdef USE_SHADOWMAP
	#if NUM_DIR_LIGHT_SHADOWS > 0
	DirectionalLightShadow directionalLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_DIR_LIGHT_SHADOWS; i ++ ) {
		directionalLight = directionalLightShadows[ i ];
		shadow *= receiveShadow ? getShadow( directionalShadowMap[ i ], directionalLight.shadowMapSize, directionalLight.shadowIntensity, directionalLight.shadowBias, directionalLight.shadowRadius, vDirectionalShadowCoord[ i ] ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
	SpotLightShadow spotLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHT_SHADOWS; i ++ ) {
		spotLight = spotLightShadows[ i ];
		shadow *= receiveShadow ? getShadow( spotShadowMap[ i ], spotLight.shadowMapSize, spotLight.shadowIntensity, spotLight.shadowBias, spotLight.shadowRadius, vSpotLightCoord[ i ] ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0 && ( defined( SHADOWMAP_TYPE_PCF ) || defined( SHADOWMAP_TYPE_BASIC ) )
	PointLightShadow pointLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_POINT_LIGHT_SHADOWS; i ++ ) {
		pointLight = pointLightShadows[ i ];
		shadow *= receiveShadow ? getPointShadow( pointShadowMap[ i ], pointLight.shadowMapSize, pointLight.shadowIntensity, pointLight.shadowBias, pointLight.shadowRadius, vPointShadowCoord[ i ], pointLight.shadowCameraNear, pointLight.shadowCameraFar ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#endif
	return shadow;
}`,rv=`#ifdef USE_SKINNING
	mat4 boneMatX = getBoneMatrix( skinIndex.x );
	mat4 boneMatY = getBoneMatrix( skinIndex.y );
	mat4 boneMatZ = getBoneMatrix( skinIndex.z );
	mat4 boneMatW = getBoneMatrix( skinIndex.w );
#endif`,sv=`#ifdef USE_SKINNING
	uniform mat4 bindMatrix;
	uniform mat4 bindMatrixInverse;
	uniform highp sampler2D boneTexture;
	mat4 getBoneMatrix( const in float i ) {
		int size = textureSize( boneTexture, 0 ).x;
		int j = int( i ) * 4;
		int x = j % size;
		int y = j / size;
		vec4 v1 = texelFetch( boneTexture, ivec2( x, y ), 0 );
		vec4 v2 = texelFetch( boneTexture, ivec2( x + 1, y ), 0 );
		vec4 v3 = texelFetch( boneTexture, ivec2( x + 2, y ), 0 );
		vec4 v4 = texelFetch( boneTexture, ivec2( x + 3, y ), 0 );
		return mat4( v1, v2, v3, v4 );
	}
#endif`,av=`#ifdef USE_SKINNING
	vec4 skinVertex = bindMatrix * vec4( transformed, 1.0 );
	vec4 skinned = vec4( 0.0 );
	skinned += boneMatX * skinVertex * skinWeight.x;
	skinned += boneMatY * skinVertex * skinWeight.y;
	skinned += boneMatZ * skinVertex * skinWeight.z;
	skinned += boneMatW * skinVertex * skinWeight.w;
	transformed = ( bindMatrixInverse * skinned ).xyz;
#endif`,ov=`#ifdef USE_SKINNING
	mat4 skinMatrix = mat4( 0.0 );
	skinMatrix += skinWeight.x * boneMatX;
	skinMatrix += skinWeight.y * boneMatY;
	skinMatrix += skinWeight.z * boneMatZ;
	skinMatrix += skinWeight.w * boneMatW;
	skinMatrix = bindMatrixInverse * skinMatrix * bindMatrix;
	objectNormal = vec4( skinMatrix * vec4( objectNormal, 0.0 ) ).xyz;
	#ifdef USE_TANGENT
		objectTangent = vec4( skinMatrix * vec4( objectTangent, 0.0 ) ).xyz;
	#endif
#endif`,lv=`float specularStrength;
#ifdef USE_SPECULARMAP
	vec4 texelSpecular = texture2D( specularMap, vSpecularMapUv );
	specularStrength = texelSpecular.r;
#else
	specularStrength = 1.0;
#endif`,cv=`#ifdef USE_SPECULARMAP
	uniform sampler2D specularMap;
#endif`,uv=`#if defined( TONE_MAPPING )
	gl_FragColor.rgb = toneMapping( gl_FragColor.rgb );
#endif`,fv=`#ifndef saturate
#define saturate( a ) clamp( a, 0.0, 1.0 )
#endif
uniform float toneMappingExposure;
vec3 LinearToneMapping( vec3 color ) {
	return saturate( toneMappingExposure * color );
}
vec3 ReinhardToneMapping( vec3 color ) {
	color *= toneMappingExposure;
	return saturate( color / ( vec3( 1.0 ) + color ) );
}
vec3 CineonToneMapping( vec3 color ) {
	color *= toneMappingExposure;
	color = max( vec3( 0.0 ), color - 0.004 );
	return pow( ( color * ( 6.2 * color + 0.5 ) ) / ( color * ( 6.2 * color + 1.7 ) + 0.06 ), vec3( 2.2 ) );
}
vec3 RRTAndODTFit( vec3 v ) {
	vec3 a = v * ( v + 0.0245786 ) - 0.000090537;
	vec3 b = v * ( 0.983729 * v + 0.4329510 ) + 0.238081;
	return a / b;
}
vec3 ACESFilmicToneMapping( vec3 color ) {
	const mat3 ACESInputMat = mat3(
		vec3( 0.59719, 0.07600, 0.02840 ),		vec3( 0.35458, 0.90834, 0.13383 ),
		vec3( 0.04823, 0.01566, 0.83777 )
	);
	const mat3 ACESOutputMat = mat3(
		vec3(  1.60475, -0.10208, -0.00327 ),		vec3( -0.53108,  1.10813, -0.07276 ),
		vec3( -0.07367, -0.00605,  1.07602 )
	);
	color *= toneMappingExposure / 0.6;
	color = ACESInputMat * color;
	color = RRTAndODTFit( color );
	color = ACESOutputMat * color;
	return saturate( color );
}
const mat3 LINEAR_REC2020_TO_LINEAR_SRGB = mat3(
	vec3( 1.6605, - 0.1246, - 0.0182 ),
	vec3( - 0.5876, 1.1329, - 0.1006 ),
	vec3( - 0.0728, - 0.0083, 1.1187 )
);
const mat3 LINEAR_SRGB_TO_LINEAR_REC2020 = mat3(
	vec3( 0.6274, 0.0691, 0.0164 ),
	vec3( 0.3293, 0.9195, 0.0880 ),
	vec3( 0.0433, 0.0113, 0.8956 )
);
vec3 agxDefaultContrastApprox( vec3 x ) {
	vec3 x2 = x * x;
	vec3 x4 = x2 * x2;
	return + 15.5 * x4 * x2
		- 40.14 * x4 * x
		+ 31.96 * x4
		- 6.868 * x2 * x
		+ 0.4298 * x2
		+ 0.1191 * x
		- 0.00232;
}
vec3 AgXToneMapping( vec3 color ) {
	const mat3 AgXInsetMatrix = mat3(
		vec3( 0.856627153315983, 0.137318972929847, 0.11189821299995 ),
		vec3( 0.0951212405381588, 0.761241990602591, 0.0767994186031903 ),
		vec3( 0.0482516061458583, 0.101439036467562, 0.811302368396859 )
	);
	const mat3 AgXOutsetMatrix = mat3(
		vec3( 1.1271005818144368, - 0.1413297634984383, - 0.14132976349843826 ),
		vec3( - 0.11060664309660323, 1.157823702216272, - 0.11060664309660294 ),
		vec3( - 0.016493938717834573, - 0.016493938717834257, 1.2519364065950405 )
	);
	const float AgxMinEv = - 12.47393;	const float AgxMaxEv = 4.026069;
	color *= toneMappingExposure;
	color = LINEAR_SRGB_TO_LINEAR_REC2020 * color;
	color = AgXInsetMatrix * color;
	color = max( color, 1e-10 );	color = log2( color );
	color = ( color - AgxMinEv ) / ( AgxMaxEv - AgxMinEv );
	color = clamp( color, 0.0, 1.0 );
	color = agxDefaultContrastApprox( color );
	color = AgXOutsetMatrix * color;
	color = pow( max( vec3( 0.0 ), color ), vec3( 2.2 ) );
	color = LINEAR_REC2020_TO_LINEAR_SRGB * color;
	color = clamp( color, 0.0, 1.0 );
	return color;
}
vec3 NeutralToneMapping( vec3 color ) {
	const float StartCompression = 0.8 - 0.04;
	const float Desaturation = 0.15;
	color *= toneMappingExposure;
	float x = min( color.r, min( color.g, color.b ) );
	float offset = x < 0.08 ? x - 6.25 * x * x : 0.04;
	color -= offset;
	float peak = max( color.r, max( color.g, color.b ) );
	if ( peak < StartCompression ) return color;
	float d = 1. - StartCompression;
	float newPeak = 1. - d * d / ( peak + d - StartCompression );
	color *= newPeak / peak;
	float g = 1. - 1. / ( Desaturation * ( peak - newPeak ) + 1. );
	return mix( color, vec3( newPeak ), g );
}
vec3 CustomToneMapping( vec3 color ) { return color; }`,hv=`#ifdef USE_TRANSMISSION
	material.transmission = transmission;
	material.transmissionAlpha = 1.0;
	material.thickness = thickness;
	material.attenuationDistance = attenuationDistance;
	material.attenuationColor = attenuationColor;
	#ifdef USE_TRANSMISSIONMAP
		material.transmission *= texture2D( transmissionMap, vTransmissionMapUv ).r;
	#endif
	#ifdef USE_THICKNESSMAP
		material.thickness *= texture2D( thicknessMap, vThicknessMapUv ).g;
	#endif
	vec3 pos = vWorldPosition;
	vec3 v = normalize( cameraPosition - pos );
	vec3 n = transformNormalByInverseViewMatrix( normal, viewMatrix );
	vec4 transmitted = getIBLVolumeRefraction(
		n, v, material.roughness, material.diffuseContribution, material.specularColorBlended, material.specularF90,
		pos, modelMatrix, viewMatrix, projectionMatrix, material.dispersion, material.ior, material.thickness,
		material.attenuationColor, material.attenuationDistance );
	material.transmissionAlpha = mix( material.transmissionAlpha, transmitted.a, material.transmission );
	totalDiffuse = mix( totalDiffuse, transmitted.rgb, material.transmission );
#endif`,dv=`#ifdef USE_TRANSMISSION
	uniform float transmission;
	uniform float thickness;
	uniform float attenuationDistance;
	uniform vec3 attenuationColor;
	#ifdef USE_TRANSMISSIONMAP
		uniform sampler2D transmissionMap;
	#endif
	#ifdef USE_THICKNESSMAP
		uniform sampler2D thicknessMap;
	#endif
	uniform vec2 transmissionSamplerSize;
	uniform sampler2D transmissionSamplerMap;
	uniform mat4 modelMatrix;
	uniform mat4 projectionMatrix;
	varying vec3 vWorldPosition;
	float w0( float a ) {
		return ( 1.0 / 6.0 ) * ( a * ( a * ( - a + 3.0 ) - 3.0 ) + 1.0 );
	}
	float w1( float a ) {
		return ( 1.0 / 6.0 ) * ( a *  a * ( 3.0 * a - 6.0 ) + 4.0 );
	}
	float w2( float a ){
		return ( 1.0 / 6.0 ) * ( a * ( a * ( - 3.0 * a + 3.0 ) + 3.0 ) + 1.0 );
	}
	float w3( float a ) {
		return ( 1.0 / 6.0 ) * ( a * a * a );
	}
	float g0( float a ) {
		return w0( a ) + w1( a );
	}
	float g1( float a ) {
		return w2( a ) + w3( a );
	}
	float h0( float a ) {
		return - 1.0 + w1( a ) / ( w0( a ) + w1( a ) );
	}
	float h1( float a ) {
		return 1.0 + w3( a ) / ( w2( a ) + w3( a ) );
	}
	vec4 bicubic( sampler2D tex, vec2 uv, vec4 texelSize, float lod ) {
		uv = uv * texelSize.zw + 0.5;
		vec2 iuv = floor( uv );
		vec2 fuv = fract( uv );
		float g0x = g0( fuv.x );
		float g1x = g1( fuv.x );
		float h0x = h0( fuv.x );
		float h1x = h1( fuv.x );
		float h0y = h0( fuv.y );
		float h1y = h1( fuv.y );
		vec2 p0 = ( vec2( iuv.x + h0x, iuv.y + h0y ) - 0.5 ) * texelSize.xy;
		vec2 p1 = ( vec2( iuv.x + h1x, iuv.y + h0y ) - 0.5 ) * texelSize.xy;
		vec2 p2 = ( vec2( iuv.x + h0x, iuv.y + h1y ) - 0.5 ) * texelSize.xy;
		vec2 p3 = ( vec2( iuv.x + h1x, iuv.y + h1y ) - 0.5 ) * texelSize.xy;
		return g0( fuv.y ) * ( g0x * textureLod( tex, p0, lod ) + g1x * textureLod( tex, p1, lod ) ) +
			g1( fuv.y ) * ( g0x * textureLod( tex, p2, lod ) + g1x * textureLod( tex, p3, lod ) );
	}
	vec4 textureBicubic( sampler2D sampler, vec2 uv, float lod ) {
		vec2 fLodSize = vec2( textureSize( sampler, int( lod ) ) );
		vec2 cLodSize = vec2( textureSize( sampler, int( lod + 1.0 ) ) );
		vec2 fLodSizeInv = 1.0 / fLodSize;
		vec2 cLodSizeInv = 1.0 / cLodSize;
		vec4 fSample = bicubic( sampler, uv, vec4( fLodSizeInv, fLodSize ), floor( lod ) );
		vec4 cSample = bicubic( sampler, uv, vec4( cLodSizeInv, cLodSize ), ceil( lod ) );
		return mix( fSample, cSample, fract( lod ) );
	}
	vec3 getVolumeTransmissionRay( const in vec3 n, const in vec3 v, const in float thickness, const in float ior, const in mat4 modelMatrix ) {
		vec3 refractionVector = refract( - v, normalize( n ), 1.0 / ior );
		vec3 modelScale;
		modelScale.x = length( vec3( modelMatrix[ 0 ].xyz ) );
		modelScale.y = length( vec3( modelMatrix[ 1 ].xyz ) );
		modelScale.z = length( vec3( modelMatrix[ 2 ].xyz ) );
		return normalize( refractionVector ) * thickness * modelScale;
	}
	float applyIorToRoughness( const in float roughness, const in float ior ) {
		return roughness * clamp( ior * 2.0 - 2.0, 0.0, 1.0 );
	}
	vec4 getTransmissionSample( const in vec2 fragCoord, const in float roughness, const in float ior ) {
		float lod = log2( transmissionSamplerSize.x ) * applyIorToRoughness( roughness, ior );
		return textureBicubic( transmissionSamplerMap, fragCoord.xy, lod );
	}
	vec3 volumeAttenuation( const in float transmissionDistance, const in vec3 attenuationColor, const in float attenuationDistance ) {
		if ( isinf( attenuationDistance ) ) {
			return vec3( 1.0 );
		} else {
			vec3 attenuationCoefficient = -log( attenuationColor ) / attenuationDistance;
			vec3 transmittance = exp( - attenuationCoefficient * transmissionDistance );			return transmittance;
		}
	}
	vec4 getIBLVolumeRefraction( const in vec3 n, const in vec3 v, const in float roughness, const in vec3 diffuseColor,
		const in vec3 specularColor, const in float specularF90, const in vec3 position, const in mat4 modelMatrix,
		const in mat4 viewMatrix, const in mat4 projMatrix, const in float dispersion, const in float ior, const in float thickness,
		const in vec3 attenuationColor, const in float attenuationDistance ) {
		vec4 transmittedLight;
		vec3 transmittance;
		#ifdef USE_DISPERSION
			float halfSpread = ( ior - 1.0 ) * 0.025 * dispersion;
			vec3 iors = vec3( ior - halfSpread, ior, ior + halfSpread );
			for ( int i = 0; i < 3; i ++ ) {
				vec3 transmissionRay = getVolumeTransmissionRay( n, v, thickness, iors[ i ], modelMatrix );
				vec3 refractedRayExit = position + transmissionRay;
				vec4 ndcPos = projMatrix * viewMatrix * vec4( refractedRayExit, 1.0 );
				vec2 refractionCoords = ndcPos.xy / ndcPos.w;
				refractionCoords += 1.0;
				refractionCoords /= 2.0;
				vec4 transmissionSample = getTransmissionSample( refractionCoords, roughness, iors[ i ] );
				transmittedLight[ i ] = transmissionSample[ i ];
				transmittedLight.a += transmissionSample.a;
				transmittance[ i ] = diffuseColor[ i ] * volumeAttenuation( length( transmissionRay ), attenuationColor, attenuationDistance )[ i ];
			}
			transmittedLight.a /= 3.0;
		#else
			vec3 transmissionRay = getVolumeTransmissionRay( n, v, thickness, ior, modelMatrix );
			vec3 refractedRayExit = position + transmissionRay;
			vec4 ndcPos = projMatrix * viewMatrix * vec4( refractedRayExit, 1.0 );
			vec2 refractionCoords = ndcPos.xy / ndcPos.w;
			refractionCoords += 1.0;
			refractionCoords /= 2.0;
			transmittedLight = getTransmissionSample( refractionCoords, roughness, ior );
			transmittance = diffuseColor * volumeAttenuation( length( transmissionRay ), attenuationColor, attenuationDistance );
		#endif
		vec3 attenuatedColor = transmittance * transmittedLight.rgb;
		vec3 F = EnvironmentBRDF( n, v, specularColor, specularF90, roughness );
		float transmittanceFactor = ( transmittance.r + transmittance.g + transmittance.b ) / 3.0;
		return vec4( ( 1.0 - F ) * attenuatedColor, 1.0 - ( 1.0 - transmittedLight.a ) * transmittanceFactor );
	}
#endif`,pv=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	varying vec2 vUv;
#endif
#ifdef USE_MAP
	varying vec2 vMapUv;
#endif
#ifdef USE_ALPHAMAP
	varying vec2 vAlphaMapUv;
#endif
#ifdef USE_LIGHTMAP
	varying vec2 vLightMapUv;
#endif
#ifdef USE_AOMAP
	varying vec2 vAoMapUv;
#endif
#ifdef USE_BUMPMAP
	varying vec2 vBumpMapUv;
#endif
#ifdef USE_NORMALMAP
	varying vec2 vNormalMapUv;
#endif
#ifdef USE_EMISSIVEMAP
	varying vec2 vEmissiveMapUv;
#endif
#ifdef USE_METALNESSMAP
	varying vec2 vMetalnessMapUv;
#endif
#ifdef USE_ROUGHNESSMAP
	varying vec2 vRoughnessMapUv;
#endif
#ifdef USE_ANISOTROPYMAP
	varying vec2 vAnisotropyMapUv;
#endif
#ifdef USE_CLEARCOATMAP
	varying vec2 vClearcoatMapUv;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	varying vec2 vClearcoatNormalMapUv;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	varying vec2 vClearcoatRoughnessMapUv;
#endif
#ifdef USE_IRIDESCENCEMAP
	varying vec2 vIridescenceMapUv;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	varying vec2 vIridescenceThicknessMapUv;
#endif
#ifdef USE_SHEEN_COLORMAP
	varying vec2 vSheenColorMapUv;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	varying vec2 vSheenRoughnessMapUv;
#endif
#ifdef USE_SPECULARMAP
	varying vec2 vSpecularMapUv;
#endif
#ifdef USE_SPECULAR_COLORMAP
	varying vec2 vSpecularColorMapUv;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	varying vec2 vSpecularIntensityMapUv;
#endif
#ifdef USE_TRANSMISSIONMAP
	uniform mat3 transmissionMapTransform;
	varying vec2 vTransmissionMapUv;
#endif
#ifdef USE_THICKNESSMAP
	uniform mat3 thicknessMapTransform;
	varying vec2 vThicknessMapUv;
#endif`,mv=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	varying vec2 vUv;
#endif
#ifdef USE_MAP
	uniform mat3 mapTransform;
	varying vec2 vMapUv;
#endif
#ifdef USE_ALPHAMAP
	uniform mat3 alphaMapTransform;
	varying vec2 vAlphaMapUv;
#endif
#ifdef USE_LIGHTMAP
	uniform mat3 lightMapTransform;
	varying vec2 vLightMapUv;
#endif
#ifdef USE_AOMAP
	uniform mat3 aoMapTransform;
	varying vec2 vAoMapUv;
#endif
#ifdef USE_BUMPMAP
	uniform mat3 bumpMapTransform;
	varying vec2 vBumpMapUv;
#endif
#ifdef USE_NORMALMAP
	uniform mat3 normalMapTransform;
	varying vec2 vNormalMapUv;
#endif
#ifdef USE_DISPLACEMENTMAP
	uniform mat3 displacementMapTransform;
	varying vec2 vDisplacementMapUv;
#endif
#ifdef USE_EMISSIVEMAP
	uniform mat3 emissiveMapTransform;
	varying vec2 vEmissiveMapUv;
#endif
#ifdef USE_METALNESSMAP
	uniform mat3 metalnessMapTransform;
	varying vec2 vMetalnessMapUv;
#endif
#ifdef USE_ROUGHNESSMAP
	uniform mat3 roughnessMapTransform;
	varying vec2 vRoughnessMapUv;
#endif
#ifdef USE_ANISOTROPYMAP
	uniform mat3 anisotropyMapTransform;
	varying vec2 vAnisotropyMapUv;
#endif
#ifdef USE_CLEARCOATMAP
	uniform mat3 clearcoatMapTransform;
	varying vec2 vClearcoatMapUv;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	uniform mat3 clearcoatNormalMapTransform;
	varying vec2 vClearcoatNormalMapUv;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	uniform mat3 clearcoatRoughnessMapTransform;
	varying vec2 vClearcoatRoughnessMapUv;
#endif
#ifdef USE_SHEEN_COLORMAP
	uniform mat3 sheenColorMapTransform;
	varying vec2 vSheenColorMapUv;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	uniform mat3 sheenRoughnessMapTransform;
	varying vec2 vSheenRoughnessMapUv;
#endif
#ifdef USE_IRIDESCENCEMAP
	uniform mat3 iridescenceMapTransform;
	varying vec2 vIridescenceMapUv;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	uniform mat3 iridescenceThicknessMapTransform;
	varying vec2 vIridescenceThicknessMapUv;
#endif
#ifdef USE_SPECULARMAP
	uniform mat3 specularMapTransform;
	varying vec2 vSpecularMapUv;
#endif
#ifdef USE_SPECULAR_COLORMAP
	uniform mat3 specularColorMapTransform;
	varying vec2 vSpecularColorMapUv;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	uniform mat3 specularIntensityMapTransform;
	varying vec2 vSpecularIntensityMapUv;
#endif
#ifdef USE_TRANSMISSIONMAP
	uniform mat3 transmissionMapTransform;
	varying vec2 vTransmissionMapUv;
#endif
#ifdef USE_THICKNESSMAP
	uniform mat3 thicknessMapTransform;
	varying vec2 vThicknessMapUv;
#endif`,gv=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	vUv = vec3( uv, 1 ).xy;
#endif
#ifdef USE_MAP
	vMapUv = ( mapTransform * vec3( MAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ALPHAMAP
	vAlphaMapUv = ( alphaMapTransform * vec3( ALPHAMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_LIGHTMAP
	vLightMapUv = ( lightMapTransform * vec3( LIGHTMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_AOMAP
	vAoMapUv = ( aoMapTransform * vec3( AOMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_BUMPMAP
	vBumpMapUv = ( bumpMapTransform * vec3( BUMPMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_NORMALMAP
	vNormalMapUv = ( normalMapTransform * vec3( NORMALMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_DISPLACEMENTMAP
	vDisplacementMapUv = ( displacementMapTransform * vec3( DISPLACEMENTMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_EMISSIVEMAP
	vEmissiveMapUv = ( emissiveMapTransform * vec3( EMISSIVEMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_METALNESSMAP
	vMetalnessMapUv = ( metalnessMapTransform * vec3( METALNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ROUGHNESSMAP
	vRoughnessMapUv = ( roughnessMapTransform * vec3( ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ANISOTROPYMAP
	vAnisotropyMapUv = ( anisotropyMapTransform * vec3( ANISOTROPYMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOATMAP
	vClearcoatMapUv = ( clearcoatMapTransform * vec3( CLEARCOATMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	vClearcoatNormalMapUv = ( clearcoatNormalMapTransform * vec3( CLEARCOAT_NORMALMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	vClearcoatRoughnessMapUv = ( clearcoatRoughnessMapTransform * vec3( CLEARCOAT_ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_IRIDESCENCEMAP
	vIridescenceMapUv = ( iridescenceMapTransform * vec3( IRIDESCENCEMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	vIridescenceThicknessMapUv = ( iridescenceThicknessMapTransform * vec3( IRIDESCENCE_THICKNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SHEEN_COLORMAP
	vSheenColorMapUv = ( sheenColorMapTransform * vec3( SHEEN_COLORMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	vSheenRoughnessMapUv = ( sheenRoughnessMapTransform * vec3( SHEEN_ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULARMAP
	vSpecularMapUv = ( specularMapTransform * vec3( SPECULARMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULAR_COLORMAP
	vSpecularColorMapUv = ( specularColorMapTransform * vec3( SPECULAR_COLORMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	vSpecularIntensityMapUv = ( specularIntensityMapTransform * vec3( SPECULAR_INTENSITYMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_TRANSMISSIONMAP
	vTransmissionMapUv = ( transmissionMapTransform * vec3( TRANSMISSIONMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_THICKNESSMAP
	vThicknessMapUv = ( thicknessMapTransform * vec3( THICKNESSMAP_UV, 1 ) ).xy;
#endif`,_v=`#if defined( USE_ENVMAP ) || defined( DISTANCE ) || defined ( USE_SHADOWMAP ) || defined ( USE_TRANSMISSION ) || NUM_SPOT_LIGHT_COORDS > 0
	vec4 worldPosition = vec4( transformed, 1.0 );
	#ifdef USE_BATCHING
		worldPosition = batchingMatrix * worldPosition;
	#endif
	#ifdef USE_INSTANCING
		worldPosition = instanceMatrix * worldPosition;
	#endif
	worldPosition = modelMatrix * worldPosition;
#endif`;const xv=`varying vec2 vUv;
uniform mat3 uvTransform;
void main() {
	vUv = ( uvTransform * vec3( uv, 1 ) ).xy;
	gl_Position = vec4( position.xy, 1.0, 1.0 );
}`,vv=`uniform sampler2D t2D;
uniform float backgroundIntensity;
varying vec2 vUv;
void main() {
	vec4 texColor = texture2D( t2D, vUv );
	#ifdef DECODE_VIDEO_TEXTURE
		texColor = vec4( mix( pow( texColor.rgb * 0.9478672986 + vec3( 0.0521327014 ), vec3( 2.4 ) ), texColor.rgb * 0.0773993808, vec3( lessThanEqual( texColor.rgb, vec3( 0.04045 ) ) ) ), texColor.w );
	#endif
	texColor.rgb *= backgroundIntensity;
	gl_FragColor = texColor;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,Sv=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,Mv=`#ifdef ENVMAP_TYPE_CUBE
	uniform samplerCube envMap;
#elif defined( ENVMAP_TYPE_CUBE_UV )
	uniform sampler2D envMap;
#endif
uniform float backgroundBlurriness;
uniform float backgroundIntensity;
uniform mat3 backgroundRotation;
varying vec3 vWorldDirection;
#include <cube_uv_reflection_fragment>
void main() {
	#ifdef ENVMAP_TYPE_CUBE
		vec4 texColor = textureCube( envMap, backgroundRotation * vWorldDirection );
	#elif defined( ENVMAP_TYPE_CUBE_UV )
		vec4 texColor = textureCubeUV( envMap, backgroundRotation * vWorldDirection, backgroundBlurriness );
	#else
		vec4 texColor = vec4( 0.0, 0.0, 0.0, 1.0 );
	#endif
	texColor.rgb *= backgroundIntensity;
	gl_FragColor = texColor;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,yv=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,Ev=`uniform samplerCube tCube;
uniform float tFlip;
uniform float opacity;
varying vec3 vWorldDirection;
void main() {
	vec4 texColor = textureCube( tCube, vec3( tFlip * vWorldDirection.x, vWorldDirection.yz ) );
	gl_FragColor = texColor;
	gl_FragColor.a *= opacity;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,bv=`#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
varying vec2 vHighPrecisionZW;
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <skinbase_vertex>
	#include <morphinstance_vertex>
	#ifdef USE_DISPLACEMENTMAP
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vHighPrecisionZW = gl_Position.zw;
}`,Tv=`#if DEPTH_PACKING == 3200
	uniform float opacity;
#endif
#include <common>
#include <packing>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
varying vec2 vHighPrecisionZW;
void main() {
	vec4 diffuseColor = vec4( 1.0 );
	#include <clipping_planes_fragment>
	#if DEPTH_PACKING == 3200
		diffuseColor.a = opacity;
	#endif
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <logdepthbuf_fragment>
	#ifdef USE_REVERSED_DEPTH_BUFFER
		float fragCoordZ = vHighPrecisionZW[ 0 ] / vHighPrecisionZW[ 1 ];
	#else
		float fragCoordZ = 0.5 * vHighPrecisionZW[ 0 ] / vHighPrecisionZW[ 1 ] + 0.5;
	#endif
	#if DEPTH_PACKING == 3200
		gl_FragColor = vec4( vec3( 1.0 - fragCoordZ ), opacity );
	#elif DEPTH_PACKING == 3201
		gl_FragColor = packDepthToRGBA( fragCoordZ );
	#elif DEPTH_PACKING == 3202
		gl_FragColor = vec4( packDepthToRGB( fragCoordZ ), 1.0 );
	#elif DEPTH_PACKING == 3203
		gl_FragColor = vec4( packDepthToRG( fragCoordZ ), 0.0, 1.0 );
	#endif
}`,Av=`#define DISTANCE
varying vec3 vWorldPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <skinbase_vertex>
	#include <morphinstance_vertex>
	#ifdef USE_DISPLACEMENTMAP
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <worldpos_vertex>
	#include <clipping_planes_vertex>
	vWorldPosition = worldPosition.xyz;
}`,wv=`#define DISTANCE
uniform vec3 referencePosition;
uniform float nearDistance;
uniform float farDistance;
varying vec3 vWorldPosition;
#include <common>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( 1.0 );
	#include <clipping_planes_fragment>
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	float dist = length( vWorldPosition - referencePosition );
	dist = ( dist - nearDistance ) / ( farDistance - nearDistance );
	dist = saturate( dist );
	gl_FragColor = vec4( dist, 0.0, 0.0, 1.0 );
}`,Rv=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
}`,Cv=`uniform sampler2D tEquirect;
varying vec3 vWorldDirection;
#include <common>
void main() {
	vec3 direction = normalize( vWorldDirection );
	vec2 sampleUV = equirectUv( direction );
	gl_FragColor = texture2D( tEquirect, sampleUV );
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,Pv=`uniform float scale;
attribute float lineDistance;
varying float vLineDistance;
#include <common>
#include <uv_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	vLineDistance = scale * lineDistance;
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
}`,Dv=`uniform vec3 diffuse;
uniform float opacity;
uniform float dashSize;
uniform float totalSize;
varying float vLineDistance;
#include <common>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	if ( mod( vLineDistance, totalSize ) > dashSize ) {
		discard;
	}
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,Lv=`#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#if defined ( USE_ENVMAP ) || defined ( USE_SKINNING )
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinbase_vertex>
		#include <skinnormal_vertex>
		#include <defaultnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <fog_vertex>
}`,Iv=`uniform vec3 diffuse;
uniform float opacity;
#ifndef FLAT_SHADED
	varying vec3 vNormal;
#endif
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <fog_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	#ifdef USE_LIGHTMAP
		vec4 lightMapTexel = texture2D( lightMap, vLightMapUv );
		reflectedLight.indirectDiffuse += lightMapTexel.rgb * lightMapIntensity * RECIPROCAL_PI;
	#else
		reflectedLight.indirectDiffuse += vec3( 1.0 );
	#endif
	#include <aomap_fragment>
	reflectedLight.indirectDiffuse *= diffuseColor.rgb;
	vec3 outgoingLight = reflectedLight.indirectDiffuse;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,Uv=`#define LAMBERT
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,Nv=`#define LAMBERT
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_lambert_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_lambert_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + totalEmissiveRadiance;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,Fv=`#define MATCAP
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <color_pars_vertex>
#include <displacementmap_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
	vViewPosition = - mvPosition.xyz;
}`,Ov=`#define MATCAP
uniform vec3 diffuse;
uniform float opacity;
uniform sampler2D matcap;
varying vec3 vViewPosition;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <normal_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	vec3 viewDir = normalize( vViewPosition );
	vec3 x = normalize( vec3( viewDir.z, 0.0, - viewDir.x ) );
	vec3 y = cross( viewDir, x );
	vec2 uv = vec2( dot( x, normal ), dot( y, normal ) ) * 0.495 + 0.5;
	#ifdef USE_MATCAP
		vec4 matcapColor = texture2D( matcap, uv );
	#else
		vec4 matcapColor = vec4( vec3( mix( 0.2, 0.8, uv.y ) ), 1.0 );
	#endif
	vec3 outgoingLight = diffuseColor.rgb * matcapColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,Bv=`#define NORMAL
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	varying vec3 vViewPosition;
#endif
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	vViewPosition = - mvPosition.xyz;
#endif
}`,zv=`#define NORMAL
uniform float opacity;
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	varying vec3 vViewPosition;
#endif
#include <uv_pars_fragment>
#include <normal_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( 0.0, 0.0, 0.0, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	gl_FragColor = vec4( normalize( normal ) * 0.5 + 0.5, diffuseColor.a );
	#ifdef OPAQUE
		gl_FragColor.a = 1.0;
	#endif
}`,kv=`#define PHONG
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,Vv=`#define PHONG
uniform vec3 diffuse;
uniform vec3 emissive;
uniform vec3 specular;
uniform float shininess;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_phong_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_phong_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + reflectedLight.directSpecular + reflectedLight.indirectSpecular + totalEmissiveRadiance;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,Gv=`#define STANDARD
varying vec3 vViewPosition;
#ifdef USE_TRANSMISSION
	varying vec3 vWorldPosition;
#endif
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
#ifdef USE_TRANSMISSION
	vWorldPosition = worldPosition.xyz;
#endif
}`,Hv=`#define STANDARD
#ifdef PHYSICAL
	#define IOR
	#define USE_SPECULAR
#endif
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float roughness;
uniform float metalness;
uniform float opacity;
#ifdef IOR
	uniform float ior;
#endif
#ifdef USE_SPECULAR
	uniform float specularIntensity;
	uniform vec3 specularColor;
	#ifdef USE_SPECULAR_COLORMAP
		uniform sampler2D specularColorMap;
	#endif
	#ifdef USE_SPECULAR_INTENSITYMAP
		uniform sampler2D specularIntensityMap;
	#endif
#endif
#ifdef USE_CLEARCOAT
	uniform float clearcoat;
	uniform float clearcoatRoughness;
#endif
#ifdef USE_DISPERSION
	uniform float dispersion;
#endif
#ifdef USE_IRIDESCENCE
	uniform float iridescence;
	uniform float iridescenceIOR;
	uniform float iridescenceThicknessMinimum;
	uniform float iridescenceThicknessMaximum;
#endif
#ifdef USE_SHEEN
	uniform vec3 sheenColor;
	uniform float sheenRoughness;
	#ifdef USE_SHEEN_COLORMAP
		uniform sampler2D sheenColorMap;
	#endif
	#ifdef USE_SHEEN_ROUGHNESSMAP
		uniform sampler2D sheenRoughnessMap;
	#endif
#endif
#ifdef USE_ANISOTROPY
	uniform vec2 anisotropyVector;
	#ifdef USE_ANISOTROPYMAP
		uniform sampler2D anisotropyMap;
	#endif
#endif
varying vec3 vViewPosition;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <iridescence_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_physical_pars_fragment>
#include <transmission_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <clearcoat_pars_fragment>
#include <iridescence_pars_fragment>
#include <roughnessmap_pars_fragment>
#include <metalnessmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <roughnessmap_fragment>
	#include <metalnessmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <clearcoat_normal_fragment_begin>
	#include <clearcoat_normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_physical_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 totalDiffuse = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse;
	vec3 totalSpecular = reflectedLight.directSpecular + reflectedLight.indirectSpecular;
	#include <transmission_fragment>
	vec3 outgoingLight = totalDiffuse + totalSpecular + totalEmissiveRadiance;
	#ifdef USE_SHEEN
 
		outgoingLight = outgoingLight + sheenSpecularDirect + sheenSpecularIndirect;
 
 	#endif
	#ifdef USE_CLEARCOAT
		float dotNVcc = saturate( dot( geometryClearcoatNormal, geometryViewDir ) );
		vec3 Fcc = F_Schlick( material.clearcoatF0, material.clearcoatF90, dotNVcc );
		outgoingLight = outgoingLight * ( 1.0 - material.clearcoat * Fcc ) + ( clearcoatSpecularDirect + clearcoatSpecularIndirect ) * material.clearcoat;
	#endif
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,Wv=`#define TOON
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,Xv=`#define TOON
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <gradientmap_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_toon_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_toon_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + totalEmissiveRadiance;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,Yv=`uniform float size;
uniform float scale;
#include <common>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
#ifdef USE_POINTS_UV
	varying vec2 vUv;
	uniform mat3 uvTransform;
#endif
void main() {
	#ifdef USE_POINTS_UV
		vUv = ( uvTransform * vec3( uv, 1 ) ).xy;
	#endif
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <project_vertex>
	gl_PointSize = size;
	#ifdef USE_SIZEATTENUATION
		bool isPerspective = isPerspectiveMatrix( projectionMatrix );
		if ( isPerspective ) gl_PointSize *= ( scale / - mvPosition.z );
	#endif
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <worldpos_vertex>
	#include <fog_vertex>
}`,qv=`uniform vec3 diffuse;
uniform float opacity;
#include <common>
#include <color_pars_fragment>
#include <map_particle_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_particle_fragment>
	#include <color_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,Kv=`#include <common>
#include <batching_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <shadowmap_pars_vertex>
void main() {
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,$v=`uniform vec3 color;
uniform float opacity;
#include <common>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <logdepthbuf_pars_fragment>
#include <shadowmap_pars_fragment>
#include <shadowmask_pars_fragment>
void main() {
	#include <logdepthbuf_fragment>
	gl_FragColor = vec4( color, opacity * ( 1.0 - getShadowMask() ) );
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,Zv=`uniform float rotation;
uniform vec2 center;
#include <common>
#include <uv_pars_vertex>
#include <fog_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	vec4 mvPosition = modelViewMatrix[ 3 ];
	vec2 scale = vec2( length( modelMatrix[ 0 ].xyz ), length( modelMatrix[ 1 ].xyz ) );
	#ifndef USE_SIZEATTENUATION
		bool isPerspective = isPerspectiveMatrix( projectionMatrix );
		if ( isPerspective ) scale *= - mvPosition.z;
	#endif
	vec2 alignedPosition = ( position.xy - ( center - vec2( 0.5 ) ) ) * scale;
	vec2 rotatedPosition;
	rotatedPosition.x = cos( rotation ) * alignedPosition.x - sin( rotation ) * alignedPosition.y;
	rotatedPosition.y = sin( rotation ) * alignedPosition.x + cos( rotation ) * alignedPosition.y;
	mvPosition.xy += rotatedPosition;
	gl_Position = projectionMatrix * mvPosition;
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
}`,Jv=`uniform vec3 diffuse;
uniform float opacity;
#include <common>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
}`,st={alphahash_fragment:x_,alphahash_pars_fragment:v_,alphamap_fragment:S_,alphamap_pars_fragment:M_,alphatest_fragment:y_,alphatest_pars_fragment:E_,aomap_fragment:b_,aomap_pars_fragment:T_,batching_pars_vertex:A_,batching_vertex:w_,begin_vertex:R_,beginnormal_vertex:C_,bsdfs:P_,iridescence_fragment:D_,bumpmap_pars_fragment:L_,clipping_planes_fragment:I_,clipping_planes_pars_fragment:U_,clipping_planes_pars_vertex:N_,clipping_planes_vertex:F_,color_fragment:O_,color_pars_fragment:B_,color_pars_vertex:z_,color_vertex:k_,common:V_,cube_uv_reflection_fragment:G_,defaultnormal_vertex:H_,displacementmap_pars_vertex:W_,displacementmap_vertex:X_,emissivemap_fragment:Y_,emissivemap_pars_fragment:q_,colorspace_fragment:K_,colorspace_pars_fragment:$_,envmap_fragment:Z_,envmap_common_pars_fragment:J_,envmap_pars_fragment:j_,envmap_pars_vertex:Q_,envmap_physical_pars_fragment:ux,envmap_vertex:ex,fog_vertex:tx,fog_pars_vertex:nx,fog_fragment:ix,fog_pars_fragment:rx,gradientmap_pars_fragment:sx,lightmap_pars_fragment:ax,lights_lambert_fragment:ox,lights_lambert_pars_fragment:lx,lights_pars_begin:cx,lights_toon_fragment:fx,lights_toon_pars_fragment:hx,lights_phong_fragment:dx,lights_phong_pars_fragment:px,lights_physical_fragment:mx,lights_physical_pars_fragment:gx,lights_fragment_begin:_x,lights_fragment_maps:xx,lights_fragment_end:vx,lightprobes_pars_fragment:Sx,logdepthbuf_fragment:Mx,logdepthbuf_pars_fragment:yx,logdepthbuf_pars_vertex:Ex,logdepthbuf_vertex:bx,map_fragment:Tx,map_pars_fragment:Ax,map_particle_fragment:wx,map_particle_pars_fragment:Rx,metalnessmap_fragment:Cx,metalnessmap_pars_fragment:Px,morphinstance_vertex:Dx,morphcolor_vertex:Lx,morphnormal_vertex:Ix,morphtarget_pars_vertex:Ux,morphtarget_vertex:Nx,normal_fragment_begin:Fx,normal_fragment_maps:Ox,normal_pars_fragment:Bx,normal_pars_vertex:zx,normal_vertex:kx,normalmap_pars_fragment:Vx,clearcoat_normal_fragment_begin:Gx,clearcoat_normal_fragment_maps:Hx,clearcoat_pars_fragment:Wx,iridescence_pars_fragment:Xx,opaque_fragment:Yx,packing:qx,premultiplied_alpha_fragment:Kx,project_vertex:$x,dithering_fragment:Zx,dithering_pars_fragment:Jx,roughnessmap_fragment:jx,roughnessmap_pars_fragment:Qx,shadowmap_pars_fragment:ev,shadowmap_pars_vertex:tv,shadowmap_vertex:nv,shadowmask_pars_fragment:iv,skinbase_vertex:rv,skinning_pars_vertex:sv,skinning_vertex:av,skinnormal_vertex:ov,specularmap_fragment:lv,specularmap_pars_fragment:cv,tonemapping_fragment:uv,tonemapping_pars_fragment:fv,transmission_fragment:hv,transmission_pars_fragment:dv,uv_pars_fragment:pv,uv_pars_vertex:mv,uv_vertex:gv,worldpos_vertex:_v,background_vert:xv,background_frag:vv,backgroundCube_vert:Sv,backgroundCube_frag:Mv,cube_vert:yv,cube_frag:Ev,depth_vert:bv,depth_frag:Tv,distance_vert:Av,distance_frag:wv,equirect_vert:Rv,equirect_frag:Cv,linedashed_vert:Pv,linedashed_frag:Dv,meshbasic_vert:Lv,meshbasic_frag:Iv,meshlambert_vert:Uv,meshlambert_frag:Nv,meshmatcap_vert:Fv,meshmatcap_frag:Ov,meshnormal_vert:Bv,meshnormal_frag:zv,meshphong_vert:kv,meshphong_frag:Vv,meshphysical_vert:Gv,meshphysical_frag:Hv,meshtoon_vert:Wv,meshtoon_frag:Xv,points_vert:Yv,points_frag:qv,shadow_vert:Kv,shadow_frag:$v,sprite_vert:Zv,sprite_frag:Jv},we={common:{diffuse:{value:new ft(16777215)},opacity:{value:1},map:{value:null},mapTransform:{value:new nt},alphaMap:{value:null},alphaMapTransform:{value:new nt},alphaTest:{value:0}},specularmap:{specularMap:{value:null},specularMapTransform:{value:new nt}},envmap:{envMap:{value:null},envMapRotation:{value:new nt},reflectivity:{value:1},ior:{value:1.5},refractionRatio:{value:.98},dfgLUT:{value:null}},aomap:{aoMap:{value:null},aoMapIntensity:{value:1},aoMapTransform:{value:new nt}},lightmap:{lightMap:{value:null},lightMapIntensity:{value:1},lightMapTransform:{value:new nt}},bumpmap:{bumpMap:{value:null},bumpMapTransform:{value:new nt},bumpScale:{value:1}},normalmap:{normalMap:{value:null},normalMapTransform:{value:new nt},normalScale:{value:new $e(1,1)}},displacementmap:{displacementMap:{value:null},displacementMapTransform:{value:new nt},displacementScale:{value:1},displacementBias:{value:0}},emissivemap:{emissiveMap:{value:null},emissiveMapTransform:{value:new nt}},metalnessmap:{metalnessMap:{value:null},metalnessMapTransform:{value:new nt}},roughnessmap:{roughnessMap:{value:null},roughnessMapTransform:{value:new nt}},gradientmap:{gradientMap:{value:null}},fog:{fogDensity:{value:25e-5},fogNear:{value:1},fogFar:{value:2e3},fogColor:{value:new ft(16777215)}},lights:{ambientLightColor:{value:[]},lightProbe:{value:[]},directionalLights:{value:[],properties:{direction:{},color:{}}},directionalLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},directionalShadowMatrix:{value:[]},spotLights:{value:[],properties:{color:{},position:{},direction:{},distance:{},coneCos:{},penumbraCos:{},decay:{}}},spotLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},spotLightMap:{value:[]},spotLightMatrix:{value:[]},pointLights:{value:[],properties:{color:{},position:{},decay:{},distance:{}}},pointLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{},shadowCameraNear:{},shadowCameraFar:{}}},pointShadowMatrix:{value:[]},hemisphereLights:{value:[],properties:{direction:{},skyColor:{},groundColor:{}}},rectAreaLights:{value:[],properties:{color:{},position:{},width:{},height:{}}},ltc_1:{value:null},ltc_2:{value:null},probesSH:{value:null},probesMin:{value:new X},probesMax:{value:new X},probesResolution:{value:new X}},points:{diffuse:{value:new ft(16777215)},opacity:{value:1},size:{value:1},scale:{value:1},map:{value:null},alphaMap:{value:null},alphaMapTransform:{value:new nt},alphaTest:{value:0},uvTransform:{value:new nt}},sprite:{diffuse:{value:new ft(16777215)},opacity:{value:1},center:{value:new $e(.5,.5)},rotation:{value:0},map:{value:null},mapTransform:{value:new nt},alphaMap:{value:null},alphaMapTransform:{value:new nt},alphaTest:{value:0}}},Ri={basic:{uniforms:Fn([we.common,we.specularmap,we.envmap,we.aomap,we.lightmap,we.fog]),vertexShader:st.meshbasic_vert,fragmentShader:st.meshbasic_frag},lambert:{uniforms:Fn([we.common,we.specularmap,we.envmap,we.aomap,we.lightmap,we.emissivemap,we.bumpmap,we.normalmap,we.displacementmap,we.fog,we.lights,{emissive:{value:new ft(0)},envMapIntensity:{value:1}}]),vertexShader:st.meshlambert_vert,fragmentShader:st.meshlambert_frag},phong:{uniforms:Fn([we.common,we.specularmap,we.envmap,we.aomap,we.lightmap,we.emissivemap,we.bumpmap,we.normalmap,we.displacementmap,we.fog,we.lights,{emissive:{value:new ft(0)},specular:{value:new ft(1118481)},shininess:{value:30},envMapIntensity:{value:1}}]),vertexShader:st.meshphong_vert,fragmentShader:st.meshphong_frag},standard:{uniforms:Fn([we.common,we.envmap,we.aomap,we.lightmap,we.emissivemap,we.bumpmap,we.normalmap,we.displacementmap,we.roughnessmap,we.metalnessmap,we.fog,we.lights,{emissive:{value:new ft(0)},roughness:{value:1},metalness:{value:0},envMapIntensity:{value:1}}]),vertexShader:st.meshphysical_vert,fragmentShader:st.meshphysical_frag},toon:{uniforms:Fn([we.common,we.aomap,we.lightmap,we.emissivemap,we.bumpmap,we.normalmap,we.displacementmap,we.gradientmap,we.fog,we.lights,{emissive:{value:new ft(0)}}]),vertexShader:st.meshtoon_vert,fragmentShader:st.meshtoon_frag},matcap:{uniforms:Fn([we.common,we.bumpmap,we.normalmap,we.displacementmap,we.fog,{matcap:{value:null}}]),vertexShader:st.meshmatcap_vert,fragmentShader:st.meshmatcap_frag},points:{uniforms:Fn([we.points,we.fog]),vertexShader:st.points_vert,fragmentShader:st.points_frag},dashed:{uniforms:Fn([we.common,we.fog,{scale:{value:1},dashSize:{value:1},totalSize:{value:2}}]),vertexShader:st.linedashed_vert,fragmentShader:st.linedashed_frag},depth:{uniforms:Fn([we.common,we.displacementmap]),vertexShader:st.depth_vert,fragmentShader:st.depth_frag},normal:{uniforms:Fn([we.common,we.bumpmap,we.normalmap,we.displacementmap,{opacity:{value:1}}]),vertexShader:st.meshnormal_vert,fragmentShader:st.meshnormal_frag},sprite:{uniforms:Fn([we.sprite,we.fog]),vertexShader:st.sprite_vert,fragmentShader:st.sprite_frag},background:{uniforms:{uvTransform:{value:new nt},t2D:{value:null},backgroundIntensity:{value:1}},vertexShader:st.background_vert,fragmentShader:st.background_frag},backgroundCube:{uniforms:{envMap:{value:null},backgroundBlurriness:{value:0},backgroundIntensity:{value:1},backgroundRotation:{value:new nt}},vertexShader:st.backgroundCube_vert,fragmentShader:st.backgroundCube_frag},cube:{uniforms:{tCube:{value:null},tFlip:{value:-1},opacity:{value:1}},vertexShader:st.cube_vert,fragmentShader:st.cube_frag},equirect:{uniforms:{tEquirect:{value:null}},vertexShader:st.equirect_vert,fragmentShader:st.equirect_frag},distance:{uniforms:Fn([we.common,we.displacementmap,{referencePosition:{value:new X},nearDistance:{value:1},farDistance:{value:1e3}}]),vertexShader:st.distance_vert,fragmentShader:st.distance_frag},shadow:{uniforms:Fn([we.lights,we.fog,{color:{value:new ft(0)},opacity:{value:1}}]),vertexShader:st.shadow_vert,fragmentShader:st.shadow_frag}};Ri.physical={uniforms:Fn([Ri.standard.uniforms,{clearcoat:{value:0},clearcoatMap:{value:null},clearcoatMapTransform:{value:new nt},clearcoatNormalMap:{value:null},clearcoatNormalMapTransform:{value:new nt},clearcoatNormalScale:{value:new $e(1,1)},clearcoatRoughness:{value:0},clearcoatRoughnessMap:{value:null},clearcoatRoughnessMapTransform:{value:new nt},dispersion:{value:0},iridescence:{value:0},iridescenceMap:{value:null},iridescenceMapTransform:{value:new nt},iridescenceIOR:{value:1.3},iridescenceThicknessMinimum:{value:100},iridescenceThicknessMaximum:{value:400},iridescenceThicknessMap:{value:null},iridescenceThicknessMapTransform:{value:new nt},sheen:{value:0},sheenColor:{value:new ft(0)},sheenColorMap:{value:null},sheenColorMapTransform:{value:new nt},sheenRoughness:{value:1},sheenRoughnessMap:{value:null},sheenRoughnessMapTransform:{value:new nt},transmission:{value:0},transmissionMap:{value:null},transmissionMapTransform:{value:new nt},transmissionSamplerSize:{value:new $e},transmissionSamplerMap:{value:null},thickness:{value:0},thicknessMap:{value:null},thicknessMapTransform:{value:new nt},attenuationDistance:{value:0},attenuationColor:{value:new ft(0)},specularColor:{value:new ft(1,1,1)},specularColorMap:{value:null},specularColorMapTransform:{value:new nt},specularIntensity:{value:1},specularIntensityMap:{value:null},specularIntensityMapTransform:{value:new nt},anisotropyVector:{value:new $e},anisotropyMap:{value:null},anisotropyMapTransform:{value:new nt}}]),vertexShader:st.meshphysical_vert,fragmentShader:st.meshphysical_frag};const no={r:0,b:0,g:0},jv=new Xt,op=new nt;op.set(-1,0,0,0,1,0,0,0,1);function Qv(i,e,t,n,r,s){const a=new ft(0);let o=r===!0?0:1,l,c,f=null,h=0,u=null;function d(T){let P=T.isScene===!0?T.background:null;if(P&&P.isTexture){const S=T.backgroundBlurriness>0;P=e.get(P,S)}return P}function x(T){let P=!1;const S=d(T);S===null?m(a,o):S&&S.isColor&&(m(S,1),P=!0);const C=i.xr.getEnvironmentBlendMode();C==="additive"?t.buffers.color.setClear(0,0,0,1,s):C==="alpha-blend"&&t.buffers.color.setClear(0,0,0,0,s),(i.autoClear||P)&&(t.buffers.depth.setTest(!0),t.buffers.depth.setMask(!0),t.buffers.color.setMask(!0),i.clear(i.autoClearColor,i.autoClearDepth,i.autoClearStencil))}function E(T,P){const S=d(P);S&&(S.isCubeTexture||S.mapping===Oo)?(c===void 0&&(c=new vi(new Ma(1,1,1),new Oi({name:"BackgroundCubeMaterial",uniforms:Us(Ri.backgroundCube.uniforms),vertexShader:Ri.backgroundCube.vertexShader,fragmentShader:Ri.backgroundCube.fragmentShader,side:Vn,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),c.geometry.deleteAttribute("normal"),c.geometry.deleteAttribute("uv"),c.onBeforeRender=function(C,y,I){this.matrixWorld.copyPosition(I.matrixWorld)},Object.defineProperty(c.material,"envMap",{get:function(){return this.uniforms.envMap.value}}),n.update(c)),c.material.uniforms.envMap.value=S,c.material.uniforms.backgroundBlurriness.value=P.backgroundBlurriness,c.material.uniforms.backgroundIntensity.value=P.backgroundIntensity,c.material.uniforms.backgroundRotation.value.setFromMatrix4(jv.makeRotationFromEuler(P.backgroundRotation)).transpose(),S.isCubeTexture&&S.isRenderTargetTexture===!1&&c.material.uniforms.backgroundRotation.value.premultiply(op),c.material.toneMapped=mt.getTransfer(S.colorSpace)!==Ct,(f!==S||h!==S.version||u!==i.toneMapping)&&(c.material.needsUpdate=!0,f=S,h=S.version,u=i.toneMapping),c.layers.enableAll(),T.unshift(c,c.geometry,c.material,0,0,null)):S&&S.isTexture&&(l===void 0&&(l=new vi(new ko(2,2),new Oi({name:"BackgroundMaterial",uniforms:Us(Ri.background.uniforms),vertexShader:Ri.background.vertexShader,fragmentShader:Ri.background.fragmentShader,side:xr,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),l.geometry.deleteAttribute("normal"),Object.defineProperty(l.material,"map",{get:function(){return this.uniforms.t2D.value}}),n.update(l)),l.material.uniforms.t2D.value=S,l.material.uniforms.backgroundIntensity.value=P.backgroundIntensity,l.material.toneMapped=mt.getTransfer(S.colorSpace)!==Ct,S.matrixAutoUpdate===!0&&S.updateMatrix(),l.material.uniforms.uvTransform.value.copy(S.matrix),(f!==S||h!==S.version||u!==i.toneMapping)&&(l.material.needsUpdate=!0,f=S,h=S.version,u=i.toneMapping),l.layers.enableAll(),T.unshift(l,l.geometry,l.material,0,0,null))}function m(T,P){T.getRGB(no,ip(i)),t.buffers.color.setClear(no.r,no.g,no.b,P,s)}function p(){c!==void 0&&(c.geometry.dispose(),c.material.dispose(),c=void 0),l!==void 0&&(l.geometry.dispose(),l.material.dispose(),l=void 0)}return{getClearColor:function(){return a},setClearColor:function(T,P=1){a.set(T),o=P,m(a,o)},getClearAlpha:function(){return o},setClearAlpha:function(T){o=T,m(a,o)},render:x,addToRenderList:E,dispose:p}}function eS(i,e){const t=i.getParameter(i.MAX_VERTEX_ATTRIBS),n={},r=u(null);let s=r,a=!1;function o(D,z,O,k,L){let N=!1;const B=h(D,k,O,z);s!==B&&(s=B,c(s.object)),N=d(D,k,O,L),N&&x(D,k,O,L),L!==null&&e.update(L,i.ELEMENT_ARRAY_BUFFER),(N||a)&&(a=!1,S(D,z,O,k),L!==null&&i.bindBuffer(i.ELEMENT_ARRAY_BUFFER,e.get(L).buffer))}function l(){return i.createVertexArray()}function c(D){return i.bindVertexArray(D)}function f(D){return i.deleteVertexArray(D)}function h(D,z,O,k){const L=k.wireframe===!0;let N=n[z.id];N===void 0&&(N={},n[z.id]=N);const B=D.isInstancedMesh===!0?D.id:0;let j=N[B];j===void 0&&(j={},N[B]=j);let ne=j[O.id];ne===void 0&&(ne={},j[O.id]=ne);let ae=ne[L];return ae===void 0&&(ae=u(l()),ne[L]=ae),ae}function u(D){const z=[],O=[],k=[];for(let L=0;L<t;L++)z[L]=0,O[L]=0,k[L]=0;return{geometry:null,program:null,wireframe:!1,newAttributes:z,enabledAttributes:O,attributeDivisors:k,object:D,attributes:{},index:null}}function d(D,z,O,k){const L=s.attributes,N=z.attributes;let B=0;const j=O.getAttributes();for(const ne in j)if(j[ne].location>=0){const fe=L[ne];let re=N[ne];if(re===void 0&&(ne==="instanceMatrix"&&D.instanceMatrix&&(re=D.instanceMatrix),ne==="instanceColor"&&D.instanceColor&&(re=D.instanceColor)),fe===void 0||fe.attribute!==re||re&&fe.data!==re.data)return!0;B++}return s.attributesNum!==B||s.index!==k}function x(D,z,O,k){const L={},N=z.attributes;let B=0;const j=O.getAttributes();for(const ne in j)if(j[ne].location>=0){let fe=N[ne];fe===void 0&&(ne==="instanceMatrix"&&D.instanceMatrix&&(fe=D.instanceMatrix),ne==="instanceColor"&&D.instanceColor&&(fe=D.instanceColor));const re={};re.attribute=fe,fe&&fe.data&&(re.data=fe.data),L[ne]=re,B++}s.attributes=L,s.attributesNum=B,s.index=k}function E(){const D=s.newAttributes;for(let z=0,O=D.length;z<O;z++)D[z]=0}function m(D){p(D,0)}function p(D,z){const O=s.newAttributes,k=s.enabledAttributes,L=s.attributeDivisors;O[D]=1,k[D]===0&&(i.enableVertexAttribArray(D),k[D]=1),L[D]!==z&&(i.vertexAttribDivisor(D,z),L[D]=z)}function T(){const D=s.newAttributes,z=s.enabledAttributes;for(let O=0,k=z.length;O<k;O++)z[O]!==D[O]&&(i.disableVertexAttribArray(O),z[O]=0)}function P(D,z,O,k,L,N,B){B===!0?i.vertexAttribIPointer(D,z,O,L,N):i.vertexAttribPointer(D,z,O,k,L,N)}function S(D,z,O,k){E();const L=k.attributes,N=O.getAttributes(),B=z.defaultAttributeValues;for(const j in N){const ne=N[j];if(ne.location>=0){let ae=L[j];if(ae===void 0&&(j==="instanceMatrix"&&D.instanceMatrix&&(ae=D.instanceMatrix),j==="instanceColor"&&D.instanceColor&&(ae=D.instanceColor)),ae!==void 0){const fe=ae.normalized,re=ae.itemSize,le=e.get(ae);if(le===void 0)continue;const Ze=le.buffer,Ke=le.type,Q=le.bytesPerElement,ue=Ke===i.INT||Ke===i.UNSIGNED_INT||ae.gpuType===su;if(ae.isInterleavedBufferAttribute){const he=ae.data,Be=he.stride,Ue=ae.offset;if(he.isInstancedInterleavedBuffer){for(let ze=0;ze<ne.locationSize;ze++)p(ne.location+ze,he.meshPerAttribute);D.isInstancedMesh!==!0&&k._maxInstanceCount===void 0&&(k._maxInstanceCount=he.meshPerAttribute*he.count)}else for(let ze=0;ze<ne.locationSize;ze++)m(ne.location+ze);i.bindBuffer(i.ARRAY_BUFFER,Ze);for(let ze=0;ze<ne.locationSize;ze++)P(ne.location+ze,re/ne.locationSize,Ke,fe,Be*Q,(Ue+re/ne.locationSize*ze)*Q,ue)}else{if(ae.isInstancedBufferAttribute){for(let he=0;he<ne.locationSize;he++)p(ne.location+he,ae.meshPerAttribute);D.isInstancedMesh!==!0&&k._maxInstanceCount===void 0&&(k._maxInstanceCount=ae.meshPerAttribute*ae.count)}else for(let he=0;he<ne.locationSize;he++)m(ne.location+he);i.bindBuffer(i.ARRAY_BUFFER,Ze);for(let he=0;he<ne.locationSize;he++)P(ne.location+he,re/ne.locationSize,Ke,fe,re*Q,re/ne.locationSize*he*Q,ue)}}else if(B!==void 0){const fe=B[j];if(fe!==void 0)switch(fe.length){case 2:i.vertexAttrib2fv(ne.location,fe);break;case 3:i.vertexAttrib3fv(ne.location,fe);break;case 4:i.vertexAttrib4fv(ne.location,fe);break;default:i.vertexAttrib1fv(ne.location,fe)}}}}T()}function C(){A();for(const D in n){const z=n[D];for(const O in z){const k=z[O];for(const L in k){const N=k[L];for(const B in N)f(N[B].object),delete N[B];delete k[L]}}delete n[D]}}function y(D){if(n[D.id]===void 0)return;const z=n[D.id];for(const O in z){const k=z[O];for(const L in k){const N=k[L];for(const B in N)f(N[B].object),delete N[B];delete k[L]}}delete n[D.id]}function I(D){for(const z in n){const O=n[z];for(const k in O){const L=O[k];if(L[D.id]===void 0)continue;const N=L[D.id];for(const B in N)f(N[B].object),delete N[B];delete L[D.id]}}}function v(D){for(const z in n){const O=n[z],k=D.isInstancedMesh===!0?D.id:0,L=O[k];if(L!==void 0){for(const N in L){const B=L[N];for(const j in B)f(B[j].object),delete B[j];delete L[N]}delete O[k],Object.keys(O).length===0&&delete n[z]}}}function A(){F(),a=!0,s!==r&&(s=r,c(s.object))}function F(){r.geometry=null,r.program=null,r.wireframe=!1}return{setup:o,reset:A,resetDefaultState:F,dispose:C,releaseStatesOfGeometry:y,releaseStatesOfObject:v,releaseStatesOfProgram:I,initAttributes:E,enableAttribute:m,disableUnusedAttributes:T}}function tS(i,e,t){let n;function r(l){n=l}function s(l,c){i.drawArrays(n,l,c),t.update(c,n,1)}function a(l,c,f){f!==0&&(i.drawArraysInstanced(n,l,c,f),t.update(c,n,f))}function o(l,c,f){if(f===0)return;e.get("WEBGL_multi_draw").multiDrawArraysWEBGL(n,l,0,c,0,f);let u=0;for(let d=0;d<f;d++)u+=c[d];t.update(u,n,1)}this.setMode=r,this.render=s,this.renderInstances=a,this.renderMultiDraw=o}function nS(i,e,t,n){let r;function s(){if(r!==void 0)return r;if(e.has("EXT_texture_filter_anisotropic")===!0){const I=e.get("EXT_texture_filter_anisotropic");r=i.getParameter(I.MAX_TEXTURE_MAX_ANISOTROPY_EXT)}else r=0;return r}function a(I){return!(I!==xi&&n.convert(I)!==i.getParameter(i.IMPLEMENTATION_COLOR_READ_FORMAT))}function o(I){const v=I===Zi&&(e.has("EXT_color_buffer_half_float")||e.has("EXT_color_buffer_float"));return!(I!==Jn&&n.convert(I)!==i.getParameter(i.IMPLEMENTATION_COLOR_READ_TYPE)&&I!==Li&&!v)}function l(I){if(I==="highp"){if(i.getShaderPrecisionFormat(i.VERTEX_SHADER,i.HIGH_FLOAT).precision>0&&i.getShaderPrecisionFormat(i.FRAGMENT_SHADER,i.HIGH_FLOAT).precision>0)return"highp";I="mediump"}return I==="mediump"&&i.getShaderPrecisionFormat(i.VERTEX_SHADER,i.MEDIUM_FLOAT).precision>0&&i.getShaderPrecisionFormat(i.FRAGMENT_SHADER,i.MEDIUM_FLOAT).precision>0?"mediump":"lowp"}let c=t.precision!==void 0?t.precision:"highp";const f=l(c);f!==c&&(Je("WebGLRenderer:",c,"not supported, using",f,"instead."),c=f);const h=t.logarithmicDepthBuffer===!0,u=t.reversedDepthBuffer===!0&&e.has("EXT_clip_control");t.reversedDepthBuffer===!0&&u===!1&&Je("WebGLRenderer: Unable to use reversed depth buffer due to missing EXT_clip_control extension. Fallback to default depth buffer.");const d=i.getParameter(i.MAX_TEXTURE_IMAGE_UNITS),x=i.getParameter(i.MAX_VERTEX_TEXTURE_IMAGE_UNITS),E=i.getParameter(i.MAX_TEXTURE_SIZE),m=i.getParameter(i.MAX_CUBE_MAP_TEXTURE_SIZE),p=i.getParameter(i.MAX_VERTEX_ATTRIBS),T=i.getParameter(i.MAX_VERTEX_UNIFORM_VECTORS),P=i.getParameter(i.MAX_VARYING_VECTORS),S=i.getParameter(i.MAX_FRAGMENT_UNIFORM_VECTORS),C=i.getParameter(i.MAX_SAMPLES),y=i.getParameter(i.SAMPLES);return{isWebGL2:!0,getMaxAnisotropy:s,getMaxPrecision:l,textureFormatReadable:a,textureTypeReadable:o,precision:c,logarithmicDepthBuffer:h,reversedDepthBuffer:u,maxTextures:d,maxVertexTextures:x,maxTextureSize:E,maxCubemapSize:m,maxAttributes:p,maxVertexUniforms:T,maxVaryings:P,maxFragmentUniforms:S,maxSamples:C,samples:y}}function iS(i){const e=this;let t=null,n=0,r=!1,s=!1;const a=new hr,o=new nt,l={value:null,needsUpdate:!1};this.uniform=l,this.numPlanes=0,this.numIntersection=0,this.init=function(h,u){const d=h.length!==0||u||n!==0||r;return r=u,n=h.length,d},this.beginShadows=function(){s=!0,f(null)},this.endShadows=function(){s=!1},this.setGlobalState=function(h,u){t=f(h,u,0)},this.setState=function(h,u,d){const x=h.clippingPlanes,E=h.clipIntersection,m=h.clipShadows,p=i.get(h);if(!r||x===null||x.length===0||s&&!m)s?f(null):c();else{const T=s?0:n,P=T*4;let S=p.clippingState||null;l.value=S,S=f(x,u,P,d);for(let C=0;C!==P;++C)S[C]=t[C];p.clippingState=S,this.numIntersection=E?this.numPlanes:0,this.numPlanes+=T}};function c(){l.value!==t&&(l.value=t,l.needsUpdate=n>0),e.numPlanes=n,e.numIntersection=0}function f(h,u,d,x){const E=h!==null?h.length:0;let m=null;if(E!==0){if(m=l.value,x!==!0||m===null){const p=d+E*4,T=u.matrixWorldInverse;o.getNormalMatrix(T),(m===null||m.length<p)&&(m=new Float32Array(p));for(let P=0,S=d;P!==E;++P,S+=4)a.copy(h[P]).applyMatrix4(T,o),a.normal.toArray(m,S),m[S+3]=a.constant}l.value=m,l.needsUpdate=!0}return e.numPlanes=E,e.numIntersection=0,m}}const mr=4,gh=[.125,.215,.35,.446,.526,.582],Ur=20,rS=256,aa=new _u,_h=new ft;let Ul=null,Nl=0,Fl=0,Ol=!1;const sS=new X;class xh{constructor(e){this._renderer=e,this._pingPongRenderTarget=null,this._lodMax=0,this._cubeSize=0,this._sizeLods=[],this._sigmas=[],this._lodMeshes=[],this._backgroundBox=null,this._cubemapMaterial=null,this._equirectMaterial=null,this._blurMaterial=null,this._ggxMaterial=null}fromScene(e,t=0,n=.1,r=100,s={}){const{size:a=256,position:o=sS}=s;Ul=this._renderer.getRenderTarget(),Nl=this._renderer.getActiveCubeFace(),Fl=this._renderer.getActiveMipmapLevel(),Ol=this._renderer.xr.enabled,this._renderer.xr.enabled=!1,this._setSize(a);const l=this._allocateTargets();return l.depthBuffer=!0,this._sceneToCubeUV(e,n,r,l,o),t>0&&this._blur(l,0,0,t),this._applyPMREM(l),this._cleanup(l),l}fromEquirectangular(e,t=null){return this._fromTexture(e,t)}fromCubemap(e,t=null){return this._fromTexture(e,t)}compileCubemapShader(){this._cubemapMaterial===null&&(this._cubemapMaterial=Mh(),this._compileMaterial(this._cubemapMaterial))}compileEquirectangularShader(){this._equirectMaterial===null&&(this._equirectMaterial=Sh(),this._compileMaterial(this._equirectMaterial))}dispose(){this._dispose(),this._cubemapMaterial!==null&&this._cubemapMaterial.dispose(),this._equirectMaterial!==null&&this._equirectMaterial.dispose(),this._backgroundBox!==null&&(this._backgroundBox.geometry.dispose(),this._backgroundBox.material.dispose())}_setSize(e){this._lodMax=Math.floor(Math.log2(e)),this._cubeSize=Math.pow(2,this._lodMax)}_dispose(){this._blurMaterial!==null&&this._blurMaterial.dispose(),this._ggxMaterial!==null&&this._ggxMaterial.dispose(),this._pingPongRenderTarget!==null&&this._pingPongRenderTarget.dispose();for(let e=0;e<this._lodMeshes.length;e++)this._lodMeshes[e].geometry.dispose()}_cleanup(e){this._renderer.setRenderTarget(Ul,Nl,Fl),this._renderer.xr.enabled=Ol,e.scissorTest=!1,vs(e,0,0,e.width,e.height)}_fromTexture(e,t){e.mapping===kr||e.mapping===Ls?this._setSize(e.image.length===0?16:e.image[0].width||e.image[0].image.width):this._setSize(e.image.width/4),Ul=this._renderer.getRenderTarget(),Nl=this._renderer.getActiveCubeFace(),Fl=this._renderer.getActiveMipmapLevel(),Ol=this._renderer.xr.enabled,this._renderer.xr.enabled=!1;const n=t||this._allocateTargets();return this._textureToCubeUV(e,n),this._applyPMREM(n),this._cleanup(n),n}_allocateTargets(){const e=3*Math.max(this._cubeSize,112),t=4*this._cubeSize,n={magFilter:yn,minFilter:yn,generateMipmaps:!1,type:Zi,format:xi,colorSpace:Mo,depthBuffer:!1},r=vh(e,t,n);if(this._pingPongRenderTarget===null||this._pingPongRenderTarget.width!==e||this._pingPongRenderTarget.height!==t){this._pingPongRenderTarget!==null&&this._dispose(),this._pingPongRenderTarget=vh(e,t,n);const{_lodMax:s}=this;({lodMeshes:this._lodMeshes,sizeLods:this._sizeLods,sigmas:this._sigmas}=aS(s)),this._blurMaterial=lS(s,e,t),this._ggxMaterial=oS(s,e,t)}return r}_compileMaterial(e){const t=new vi(new gn,e);this._renderer.compile(t,aa)}_sceneToCubeUV(e,t,n,r,s){const l=new oi(90,1,t,n),c=[1,-1,1,1,1,1],f=[1,1,1,-1,-1,-1],h=this._renderer,u=h.autoClear,d=h.toneMapping;h.getClearColor(_h),h.toneMapping=Ui,h.autoClear=!1,h.state.buffers.depth.getReversed()&&(h.setRenderTarget(r),h.clearDepth(),h.setRenderTarget(null)),this._backgroundBox===null&&(this._backgroundBox=new vi(new Ma,new mu({name:"PMREM.Background",side:Vn,depthWrite:!1,depthTest:!1})));const E=this._backgroundBox,m=E.material;let p=!1;const T=e.background;T?T.isColor&&(m.color.copy(T),e.background=null,p=!0):(m.color.copy(_h),p=!0);for(let P=0;P<6;P++){const S=P%3;S===0?(l.up.set(0,c[P],0),l.position.set(s.x,s.y,s.z),l.lookAt(s.x+f[P],s.y,s.z)):S===1?(l.up.set(0,0,c[P]),l.position.set(s.x,s.y,s.z),l.lookAt(s.x,s.y+f[P],s.z)):(l.up.set(0,c[P],0),l.position.set(s.x,s.y,s.z),l.lookAt(s.x,s.y,s.z+f[P]));const C=this._cubeSize;vs(r,S*C,P>2?C:0,C,C),h.setRenderTarget(r),p&&h.render(E,l),h.render(e,l)}h.toneMapping=d,h.autoClear=u,e.background=T}_textureToCubeUV(e,t){const n=this._renderer,r=e.mapping===kr||e.mapping===Ls;r?(this._cubemapMaterial===null&&(this._cubemapMaterial=Mh()),this._cubemapMaterial.uniforms.flipEnvMap.value=e.isRenderTargetTexture===!1?-1:1):this._equirectMaterial===null&&(this._equirectMaterial=Sh());const s=r?this._cubemapMaterial:this._equirectMaterial,a=this._lodMeshes[0];a.material=s;const o=s.uniforms;o.envMap.value=e;const l=this._cubeSize;vs(t,0,0,3*l,2*l),n.setRenderTarget(t),n.render(a,aa)}_applyPMREM(e){const t=this._renderer,n=t.autoClear;t.autoClear=!1;const r=this._lodMeshes.length;for(let s=1;s<r;s++)this._applyGGXFilter(e,s-1,s);t.autoClear=n}_applyGGXFilter(e,t,n){const r=this._renderer,s=this._pingPongRenderTarget,a=this._ggxMaterial,o=this._lodMeshes[n];o.material=a;const l=a.uniforms,c=n/(this._lodMeshes.length-1),f=t/(this._lodMeshes.length-1),h=Math.sqrt(c*c-f*f),u=0+c*1.25,d=h*u,{_lodMax:x}=this,E=this._sizeLods[n],m=3*E*(n>x-mr?n-x+mr:0),p=4*(this._cubeSize-E);l.envMap.value=e.texture,l.roughness.value=d,l.mipInt.value=x-t,vs(s,m,p,3*E,2*E),r.setRenderTarget(s),r.render(o,aa),l.envMap.value=s.texture,l.roughness.value=0,l.mipInt.value=x-n,vs(e,m,p,3*E,2*E),r.setRenderTarget(e),r.render(o,aa)}_blur(e,t,n,r,s){const a=this._pingPongRenderTarget;this._halfBlur(e,a,t,n,r,"latitudinal",s),this._halfBlur(a,e,n,n,r,"longitudinal",s)}_halfBlur(e,t,n,r,s,a,o){const l=this._renderer,c=this._blurMaterial;a!=="latitudinal"&&a!=="longitudinal"&&gt("blur direction must be either latitudinal or longitudinal!");const f=3,h=this._lodMeshes[r];h.material=c;const u=c.uniforms,d=this._sizeLods[n]-1,x=isFinite(s)?Math.PI/(2*d):2*Math.PI/(2*Ur-1),E=s/x,m=isFinite(s)?1+Math.floor(f*E):Ur;m>Ur&&Je(`sigmaRadians, ${s}, is too large and will clip, as it requested ${m} samples when the maximum is set to ${Ur}`);const p=[];let T=0;for(let I=0;I<Ur;++I){const v=I/E,A=Math.exp(-v*v/2);p.push(A),I===0?T+=A:I<m&&(T+=2*A)}for(let I=0;I<p.length;I++)p[I]=p[I]/T;u.envMap.value=e.texture,u.samples.value=m,u.weights.value=p,u.latitudinal.value=a==="latitudinal",o&&(u.poleAxis.value=o);const{_lodMax:P}=this;u.dTheta.value=x,u.mipInt.value=P-n;const S=this._sizeLods[r],C=3*S*(r>P-mr?r-P+mr:0),y=4*(this._cubeSize-S);vs(t,C,y,3*S,2*S),l.setRenderTarget(t),l.render(h,aa)}}function aS(i){const e=[],t=[],n=[];let r=i;const s=i-mr+1+gh.length;for(let a=0;a<s;a++){const o=Math.pow(2,r);e.push(o);let l=1/o;a>i-mr?l=gh[a-i+mr-1]:a===0&&(l=0),t.push(l);const c=1/(o-2),f=-c,h=1+c,u=[f,f,h,f,h,h,f,f,h,h,f,h],d=6,x=6,E=3,m=2,p=1,T=new Float32Array(E*x*d),P=new Float32Array(m*x*d),S=new Float32Array(p*x*d);for(let y=0;y<d;y++){const I=y%3*2/3-1,v=y>2?0:-1,A=[I,v,0,I+2/3,v,0,I+2/3,v+1,0,I,v,0,I+2/3,v+1,0,I,v+1,0];T.set(A,E*x*y),P.set(u,m*x*y);const F=[y,y,y,y,y,y];S.set(F,p*x*y)}const C=new gn;C.setAttribute("position",new Qn(T,E)),C.setAttribute("uv",new Qn(P,m)),C.setAttribute("faceIndex",new Qn(S,p)),n.push(new vi(C,null)),r>mr&&r--}return{lodMeshes:n,sizeLods:e,sigmas:t}}function vh(i,e,t){const n=new Ni(i,e,t);return n.texture.mapping=Oo,n.texture.name="PMREM.cubeUv",n.scissorTest=!0,n}function vs(i,e,t,n,r){i.viewport.set(e,t,n,r),i.scissor.set(e,t,n,r)}function oS(i,e,t){return new Oi({name:"PMREMGGXConvolution",defines:{GGX_SAMPLES:rS,CUBEUV_TEXEL_WIDTH:1/e,CUBEUV_TEXEL_HEIGHT:1/t,CUBEUV_MAX_MIP:`${i}.0`},uniforms:{envMap:{value:null},roughness:{value:0},mipInt:{value:0}},vertexShader:Vo(),fragmentShader:`

			precision highp float;
			precision highp int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;
			uniform float roughness;
			uniform float mipInt;

			#define ENVMAP_TYPE_CUBE_UV
			#include <cube_uv_reflection_fragment>

			#define PI 3.14159265359

			// Van der Corput radical inverse
			float radicalInverse_VdC(uint bits) {
				bits = (bits << 16u) | (bits >> 16u);
				bits = ((bits & 0x55555555u) << 1u) | ((bits & 0xAAAAAAAAu) >> 1u);
				bits = ((bits & 0x33333333u) << 2u) | ((bits & 0xCCCCCCCCu) >> 2u);
				bits = ((bits & 0x0F0F0F0Fu) << 4u) | ((bits & 0xF0F0F0F0u) >> 4u);
				bits = ((bits & 0x00FF00FFu) << 8u) | ((bits & 0xFF00FF00u) >> 8u);
				return float(bits) * 2.3283064365386963e-10; // / 0x100000000
			}

			// Hammersley sequence
			vec2 hammersley(uint i, uint N) {
				return vec2(float(i) / float(N), radicalInverse_VdC(i));
			}

			// GGX VNDF importance sampling (Eric Heitz 2018)
			// "Sampling the GGX Distribution of Visible Normals"
			// https://jcgt.org/published/0007/04/01/
			vec3 importanceSampleGGX_VNDF(vec2 Xi, vec3 V, float roughness) {
				float alpha = roughness * roughness;

				// Section 4.1: Orthonormal basis
				vec3 T1 = vec3(1.0, 0.0, 0.0);
				vec3 T2 = cross(V, T1);

				// Section 4.2: Parameterization of projected area
				float r = sqrt(Xi.x);
				float phi = 2.0 * PI * Xi.y;
				float t1 = r * cos(phi);
				float t2 = r * sin(phi);
				float s = 0.5 * (1.0 + V.z);
				t2 = (1.0 - s) * sqrt(1.0 - t1 * t1) + s * t2;

				// Section 4.3: Reprojection onto hemisphere
				vec3 Nh = t1 * T1 + t2 * T2 + sqrt(max(0.0, 1.0 - t1 * t1 - t2 * t2)) * V;

				// Section 3.4: Transform back to ellipsoid configuration
				return normalize(vec3(alpha * Nh.x, alpha * Nh.y, max(0.0, Nh.z)));
			}

			void main() {
				vec3 N = normalize(vOutputDirection);
				vec3 V = N; // Assume view direction equals normal for pre-filtering

				vec3 prefilteredColor = vec3(0.0);
				float totalWeight = 0.0;

				// For very low roughness, just sample the environment directly
				if (roughness < 0.001) {
					gl_FragColor = vec4(bilinearCubeUV(envMap, N, mipInt), 1.0);
					return;
				}

				// Tangent space basis for VNDF sampling
				vec3 up = abs(N.z) < 0.999 ? vec3(0.0, 0.0, 1.0) : vec3(1.0, 0.0, 0.0);
				vec3 tangent = normalize(cross(up, N));
				vec3 bitangent = cross(N, tangent);

				for(uint i = 0u; i < uint(GGX_SAMPLES); i++) {
					vec2 Xi = hammersley(i, uint(GGX_SAMPLES));

					// For PMREM, V = N, so in tangent space V is always (0, 0, 1)
					vec3 H_tangent = importanceSampleGGX_VNDF(Xi, vec3(0.0, 0.0, 1.0), roughness);

					// Transform H back to world space
					vec3 H = normalize(tangent * H_tangent.x + bitangent * H_tangent.y + N * H_tangent.z);
					vec3 L = normalize(2.0 * dot(V, H) * H - V);

					float NdotL = max(dot(N, L), 0.0);

					if(NdotL > 0.0) {
						// Sample environment at fixed mip level
						// VNDF importance sampling handles the distribution filtering
						vec3 sampleColor = bilinearCubeUV(envMap, L, mipInt);

						// Weight by NdotL for the split-sum approximation
						// VNDF PDF naturally accounts for the visible microfacet distribution
						prefilteredColor += sampleColor * NdotL;
						totalWeight += NdotL;
					}
				}

				if (totalWeight > 0.0) {
					prefilteredColor = prefilteredColor / totalWeight;
				}

				gl_FragColor = vec4(prefilteredColor, 1.0);
			}
		`,blending:Ki,depthTest:!1,depthWrite:!1})}function lS(i,e,t){const n=new Float32Array(Ur),r=new X(0,1,0);return new Oi({name:"SphericalGaussianBlur",defines:{n:Ur,CUBEUV_TEXEL_WIDTH:1/e,CUBEUV_TEXEL_HEIGHT:1/t,CUBEUV_MAX_MIP:`${i}.0`},uniforms:{envMap:{value:null},samples:{value:1},weights:{value:n},latitudinal:{value:!1},dTheta:{value:0},mipInt:{value:0},poleAxis:{value:r}},vertexShader:Vo(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;
			uniform int samples;
			uniform float weights[ n ];
			uniform bool latitudinal;
			uniform float dTheta;
			uniform float mipInt;
			uniform vec3 poleAxis;

			#define ENVMAP_TYPE_CUBE_UV
			#include <cube_uv_reflection_fragment>

			vec3 getSample( float theta, vec3 axis ) {

				float cosTheta = cos( theta );
				// Rodrigues' axis-angle rotation
				vec3 sampleDirection = vOutputDirection * cosTheta
					+ cross( axis, vOutputDirection ) * sin( theta )
					+ axis * dot( axis, vOutputDirection ) * ( 1.0 - cosTheta );

				return bilinearCubeUV( envMap, sampleDirection, mipInt );

			}

			void main() {

				vec3 axis = latitudinal ? poleAxis : cross( poleAxis, vOutputDirection );

				if ( all( equal( axis, vec3( 0.0 ) ) ) ) {

					axis = vec3( vOutputDirection.z, 0.0, - vOutputDirection.x );

				}

				axis = normalize( axis );

				gl_FragColor = vec4( 0.0, 0.0, 0.0, 1.0 );
				gl_FragColor.rgb += weights[ 0 ] * getSample( 0.0, axis );

				for ( int i = 1; i < n; i++ ) {

					if ( i >= samples ) {

						break;

					}

					float theta = dTheta * float( i );
					gl_FragColor.rgb += weights[ i ] * getSample( -1.0 * theta, axis );
					gl_FragColor.rgb += weights[ i ] * getSample( theta, axis );

				}

			}
		`,blending:Ki,depthTest:!1,depthWrite:!1})}function Sh(){return new Oi({name:"EquirectangularToCubeUV",uniforms:{envMap:{value:null}},vertexShader:Vo(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;

			#include <common>

			void main() {

				vec3 outputDirection = normalize( vOutputDirection );
				vec2 uv = equirectUv( outputDirection );

				gl_FragColor = vec4( texture2D ( envMap, uv ).rgb, 1.0 );

			}
		`,blending:Ki,depthTest:!1,depthWrite:!1})}function Mh(){return new Oi({name:"CubemapToCubeUV",uniforms:{envMap:{value:null},flipEnvMap:{value:-1}},vertexShader:Vo(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			uniform float flipEnvMap;

			varying vec3 vOutputDirection;

			uniform samplerCube envMap;

			void main() {

				gl_FragColor = textureCube( envMap, vec3( flipEnvMap * vOutputDirection.x, vOutputDirection.yz ) );

			}
		`,blending:Ki,depthTest:!1,depthWrite:!1})}function Vo(){return`

		precision mediump float;
		precision mediump int;

		attribute float faceIndex;

		varying vec3 vOutputDirection;

		// RH coordinate system; PMREM face-indexing convention
		vec3 getDirection( vec2 uv, float face ) {

			uv = 2.0 * uv - 1.0;

			vec3 direction = vec3( uv, 1.0 );

			if ( face == 0.0 ) {

				direction = direction.zyx; // ( 1, v, u ) pos x

			} else if ( face == 1.0 ) {

				direction = direction.xzy;
				direction.xz *= -1.0; // ( -u, 1, -v ) pos y

			} else if ( face == 2.0 ) {

				direction.x *= -1.0; // ( -u, v, 1 ) pos z

			} else if ( face == 3.0 ) {

				direction = direction.zyx;
				direction.xz *= -1.0; // ( -1, v, -u ) neg x

			} else if ( face == 4.0 ) {

				direction = direction.xzy;
				direction.xy *= -1.0; // ( -u, -1, v ) neg y

			} else if ( face == 5.0 ) {

				direction.z *= -1.0; // ( u, v, -1 ) neg z

			}

			return direction;

		}

		void main() {

			vOutputDirection = getDirection( uv, faceIndex );
			gl_Position = vec4( position, 1.0 );

		}
	`}class lp extends Ni{constructor(e=1,t={}){super(e,e,t),this.isWebGLCubeRenderTarget=!0;const n={width:e,height:e,depth:1},r=[n,n,n,n,n,n];this.texture=new tp(r),this._setTextureOptions(t),this.texture.isRenderTargetTexture=!0}fromEquirectangularTexture(e,t){this.texture.type=t.type,this.texture.colorSpace=t.colorSpace,this.texture.generateMipmaps=t.generateMipmaps,this.texture.minFilter=t.minFilter,this.texture.magFilter=t.magFilter;const n={uniforms:{tEquirect:{value:null}},vertexShader:`

				varying vec3 vWorldDirection;

				vec3 transformDirection( in vec3 dir, in mat4 matrix ) {

					return normalize( ( matrix * vec4( dir, 0.0 ) ).xyz );

				}

				void main() {

					vWorldDirection = transformDirection( position, modelMatrix );

					#include <begin_vertex>
					#include <project_vertex>

				}
			`,fragmentShader:`

				uniform sampler2D tEquirect;

				varying vec3 vWorldDirection;

				#include <common>

				void main() {

					vec3 direction = normalize( vWorldDirection );

					vec2 sampleUV = equirectUv( direction );

					gl_FragColor = texture2D( tEquirect, sampleUV );

				}
			`},r=new Ma(5,5,5),s=new Oi({name:"CubemapFromEquirect",uniforms:Us(n.uniforms),vertexShader:n.vertexShader,fragmentShader:n.fragmentShader,side:Vn,blending:Ki});s.uniforms.tEquirect.value=t;const a=new vi(r,s),o=t.minFilter;return t.minFilter===Fr&&(t.minFilter=yn),new f_(1,10,this).update(e,a),t.minFilter=o,a.geometry.dispose(),a.material.dispose(),this}clear(e,t=!0,n=!0,r=!0){const s=e.getRenderTarget();for(let a=0;a<6;a++)e.setRenderTarget(this,a),e.clear(t,n,r);e.setRenderTarget(s)}}function cS(i){let e=new WeakMap,t=new WeakMap,n=null;function r(u,d=!1){return u==null?null:d?a(u):s(u)}function s(u){if(u&&u.isTexture){const d=u.mapping;if(d===sl||d===al)if(e.has(u)){const x=e.get(u).texture;return o(x,u.mapping)}else{const x=u.image;if(x&&x.height>0){const E=new lp(x.height);return E.fromEquirectangularTexture(i,u),e.set(u,E),u.addEventListener("dispose",c),o(E.texture,u.mapping)}else return null}}return u}function a(u){if(u&&u.isTexture){const d=u.mapping,x=d===sl||d===al,E=d===kr||d===Ls;if(x||E){let m=t.get(u);const p=m!==void 0?m.texture.pmremVersion:0;if(u.isRenderTargetTexture&&u.pmremVersion!==p)return n===null&&(n=new xh(i)),m=x?n.fromEquirectangular(u,m):n.fromCubemap(u,m),m.texture.pmremVersion=u.pmremVersion,t.set(u,m),m.texture;if(m!==void 0)return m.texture;{const T=u.image;return x&&T&&T.height>0||E&&T&&l(T)?(n===null&&(n=new xh(i)),m=x?n.fromEquirectangular(u):n.fromCubemap(u),m.texture.pmremVersion=u.pmremVersion,t.set(u,m),u.addEventListener("dispose",f),m.texture):null}}}return u}function o(u,d){return d===sl?u.mapping=kr:d===al&&(u.mapping=Ls),u}function l(u){let d=0;const x=6;for(let E=0;E<x;E++)u[E]!==void 0&&d++;return d===x}function c(u){const d=u.target;d.removeEventListener("dispose",c);const x=e.get(d);x!==void 0&&(e.delete(d),x.dispose())}function f(u){const d=u.target;d.removeEventListener("dispose",f);const x=t.get(d);x!==void 0&&(t.delete(d),x.dispose())}function h(){e=new WeakMap,t=new WeakMap,n!==null&&(n.dispose(),n=null)}return{get:r,dispose:h}}function uS(i){const e={};function t(n){if(e[n]!==void 0)return e[n];const r=i.getExtension(n);return e[n]=r,r}return{has:function(n){return t(n)!==null},init:function(){t("EXT_color_buffer_float"),t("WEBGL_clip_cull_distance"),t("OES_texture_float_linear"),t("EXT_color_buffer_half_float"),t("WEBGL_multisampled_render_to_texture"),t("WEBGL_render_shared_exponent")},get:function(n){const r=t(n);return r===null&&Ts("WebGLRenderer: "+n+" extension not supported."),r}}}function fS(i,e,t,n){const r={},s=new WeakMap;function a(h){const u=h.target;u.index!==null&&e.remove(u.index);for(const x in u.attributes)e.remove(u.attributes[x]);u.removeEventListener("dispose",a),delete r[u.id];const d=s.get(u);d&&(e.remove(d),s.delete(u)),n.releaseStatesOfGeometry(u),u.isInstancedBufferGeometry===!0&&delete u._maxInstanceCount,t.memory.geometries--}function o(h,u){return r[u.id]===!0||(u.addEventListener("dispose",a),r[u.id]=!0,t.memory.geometries++),u}function l(h){const u=h.attributes;for(const d in u)e.update(u[d],i.ARRAY_BUFFER)}function c(h){const u=[],d=h.index,x=h.attributes.position;let E=0;if(x===void 0)return;if(d!==null){const T=d.array;E=d.version;for(let P=0,S=T.length;P<S;P+=3){const C=T[P+0],y=T[P+1],I=T[P+2];u.push(C,y,y,I,I,C)}}else{const T=x.array;E=x.version;for(let P=0,S=T.length/3-1;P<S;P+=3){const C=P+0,y=P+1,I=P+2;u.push(C,y,y,I,I,C)}}const m=new(x.count>=65535?Jd:Zd)(u,1);m.version=E;const p=s.get(h);p&&e.remove(p),s.set(h,m)}function f(h){const u=s.get(h);if(u){const d=h.index;d!==null&&u.version<d.version&&c(h)}else c(h);return s.get(h)}return{get:o,update:l,getWireframeAttribute:f}}function hS(i,e,t){let n;function r(h){n=h}let s,a;function o(h){s=h.type,a=h.bytesPerElement}function l(h,u){i.drawElements(n,u,s,h*a),t.update(u,n,1)}function c(h,u,d){d!==0&&(i.drawElementsInstanced(n,u,s,h*a,d),t.update(u,n,d))}function f(h,u,d){if(d===0)return;e.get("WEBGL_multi_draw").multiDrawElementsWEBGL(n,u,0,s,h,0,d);let E=0;for(let m=0;m<d;m++)E+=u[m];t.update(E,n,1)}this.setMode=r,this.setIndex=o,this.render=l,this.renderInstances=c,this.renderMultiDraw=f}function dS(i){const e={geometries:0,textures:0},t={frame:0,calls:0,triangles:0,points:0,lines:0};function n(s,a,o){switch(t.calls++,a){case i.TRIANGLES:t.triangles+=o*(s/3);break;case i.LINES:t.lines+=o*(s/2);break;case i.LINE_STRIP:t.lines+=o*(s-1);break;case i.LINE_LOOP:t.lines+=o*s;break;case i.POINTS:t.points+=o*s;break;default:gt("WebGLInfo: Unknown draw mode:",a);break}}function r(){t.calls=0,t.triangles=0,t.points=0,t.lines=0}return{memory:e,render:t,programs:null,autoReset:!0,reset:r,update:n}}function pS(i,e,t){const n=new WeakMap,r=new $t;function s(a,o,l){const c=a.morphTargetInfluences,f=o.morphAttributes.position||o.morphAttributes.normal||o.morphAttributes.color,h=f!==void 0?f.length:0;let u=n.get(o);if(u===void 0||u.count!==h){let A=function(){I.dispose(),n.delete(o),o.removeEventListener("dispose",A)};u!==void 0&&u.texture.dispose();const d=o.morphAttributes.position!==void 0,x=o.morphAttributes.normal!==void 0,E=o.morphAttributes.color!==void 0,m=o.morphAttributes.position||[],p=o.morphAttributes.normal||[],T=o.morphAttributes.color||[];let P=0;d===!0&&(P=1),x===!0&&(P=2),E===!0&&(P=3);let S=o.attributes.position.count*P,C=1;S>e.maxTextureSize&&(C=Math.ceil(S/e.maxTextureSize),S=e.maxTextureSize);const y=new Float32Array(S*C*4*h),I=new qd(y,S,C,h);I.type=Li,I.needsUpdate=!0;const v=P*4;for(let F=0;F<h;F++){const D=m[F],z=p[F],O=T[F],k=S*C*4*F;for(let L=0;L<D.count;L++){const N=L*v;d===!0&&(r.fromBufferAttribute(D,L),y[k+N+0]=r.x,y[k+N+1]=r.y,y[k+N+2]=r.z,y[k+N+3]=0),x===!0&&(r.fromBufferAttribute(z,L),y[k+N+4]=r.x,y[k+N+5]=r.y,y[k+N+6]=r.z,y[k+N+7]=0),E===!0&&(r.fromBufferAttribute(O,L),y[k+N+8]=r.x,y[k+N+9]=r.y,y[k+N+10]=r.z,y[k+N+11]=O.itemSize===4?r.w:1)}}u={count:h,texture:I,size:new $e(S,C)},n.set(o,u),o.addEventListener("dispose",A)}if(a.isInstancedMesh===!0&&a.morphTexture!==null)l.getUniforms().setValue(i,"morphTexture",a.morphTexture,t);else{let d=0;for(let E=0;E<c.length;E++)d+=c[E];const x=o.morphTargetsRelative?1:1-d;l.getUniforms().setValue(i,"morphTargetBaseInfluence",x),l.getUniforms().setValue(i,"morphTargetInfluences",c)}l.getUniforms().setValue(i,"morphTargetsTexture",u.texture,t),l.getUniforms().setValue(i,"morphTargetsTextureSize",u.size)}return{update:s}}function mS(i,e,t,n,r){let s=new WeakMap;function a(c){const f=r.render.frame,h=c.geometry,u=e.get(c,h);if(s.get(u)!==f&&(e.update(u),s.set(u,f)),c.isInstancedMesh&&(c.hasEventListener("dispose",l)===!1&&c.addEventListener("dispose",l),s.get(c)!==f&&(t.update(c.instanceMatrix,i.ARRAY_BUFFER),c.instanceColor!==null&&t.update(c.instanceColor,i.ARRAY_BUFFER),s.set(c,f))),c.isSkinnedMesh){const d=c.skeleton;s.get(d)!==f&&(d.update(),s.set(d,f))}return u}function o(){s=new WeakMap}function l(c){const f=c.target;f.removeEventListener("dispose",l),n.releaseStatesOfObject(f),t.remove(f.instanceMatrix),f.instanceColor!==null&&t.remove(f.instanceColor)}return{update:a,dispose:o}}const gS={[Dd]:"LINEAR_TONE_MAPPING",[Ld]:"REINHARD_TONE_MAPPING",[Id]:"CINEON_TONE_MAPPING",[Ud]:"ACES_FILMIC_TONE_MAPPING",[Fd]:"AGX_TONE_MAPPING",[Od]:"NEUTRAL_TONE_MAPPING",[Nd]:"CUSTOM_TONE_MAPPING"};function _S(i,e,t,n,r,s){const a=new Ni(e,t,{type:i,depthBuffer:r,stencilBuffer:s,samples:n?4:0,depthTexture:r?new Is(e,t):void 0}),o=new Ni(e,t,{type:Zi,depthBuffer:!1,stencilBuffer:!1}),l=new gn;l.setAttribute("position",new Bn([-1,3,0,-1,-1,0,3,-1,0],3)),l.setAttribute("uv",new Bn([0,2,0,0,2,0],2));const c=new s_({uniforms:{tDiffuse:{value:null}},vertexShader:`
			precision highp float;

			uniform mat4 modelViewMatrix;
			uniform mat4 projectionMatrix;

			attribute vec3 position;
			attribute vec2 uv;

			varying vec2 vUv;

			void main() {
				vUv = uv;
				gl_Position = projectionMatrix * modelViewMatrix * vec4( position, 1.0 );
			}`,fragmentShader:`
			precision highp float;

			uniform sampler2D tDiffuse;

			varying vec2 vUv;

			#include <tonemapping_pars_fragment>
			#include <colorspace_pars_fragment>

			void main() {
				gl_FragColor = texture2D( tDiffuse, vUv );

				#ifdef LINEAR_TONE_MAPPING
					gl_FragColor.rgb = LinearToneMapping( gl_FragColor.rgb );
				#elif defined( REINHARD_TONE_MAPPING )
					gl_FragColor.rgb = ReinhardToneMapping( gl_FragColor.rgb );
				#elif defined( CINEON_TONE_MAPPING )
					gl_FragColor.rgb = CineonToneMapping( gl_FragColor.rgb );
				#elif defined( ACES_FILMIC_TONE_MAPPING )
					gl_FragColor.rgb = ACESFilmicToneMapping( gl_FragColor.rgb );
				#elif defined( AGX_TONE_MAPPING )
					gl_FragColor.rgb = AgXToneMapping( gl_FragColor.rgb );
				#elif defined( NEUTRAL_TONE_MAPPING )
					gl_FragColor.rgb = NeutralToneMapping( gl_FragColor.rgb );
				#elif defined( CUSTOM_TONE_MAPPING )
					gl_FragColor.rgb = CustomToneMapping( gl_FragColor.rgb );
				#endif

				#ifdef SRGB_TRANSFER
					gl_FragColor = sRGBTransferOETF( gl_FragColor );
				#endif
			}`,depthTest:!1,depthWrite:!1}),f=new vi(l,c),h=new _u(-1,1,1,-1,0,1);let u=null,d=null,x=!1,E,m=null,p=[],T=!1;this.setSize=function(P,S){a.setSize(P,S),o.setSize(P,S);for(let C=0;C<p.length;C++){const y=p[C];y.setSize&&y.setSize(P,S)}},this.setEffects=function(P){p=P,T=p.length>0&&p[0].isRenderPass===!0;const S=a.width,C=a.height;for(let y=0;y<p.length;y++){const I=p[y];I.setSize&&I.setSize(S,C)}},this.begin=function(P,S){if(x||P.toneMapping===Ui&&p.length===0)return!1;if(m=S,S!==null){const C=S.width,y=S.height;(a.width!==C||a.height!==y)&&this.setSize(C,y)}return T===!1&&P.setRenderTarget(a),E=P.toneMapping,P.toneMapping=Ui,!0},this.hasRenderPass=function(){return T},this.end=function(P,S){P.toneMapping=E,x=!0;let C=a,y=o;for(let I=0;I<p.length;I++){const v=p[I];if(v.enabled!==!1&&(v.render(P,y,C,S),v.needsSwap!==!1)){const A=C;C=y,y=A}}if(u!==P.outputColorSpace||d!==P.toneMapping){u=P.outputColorSpace,d=P.toneMapping,c.defines={},mt.getTransfer(u)===Ct&&(c.defines.SRGB_TRANSFER="");const I=gS[d];I&&(c.defines[I]=""),c.needsUpdate=!0}c.uniforms.tDiffuse.value=C.texture,P.setRenderTarget(m),P.render(f,h),m=null,x=!1},this.isCompositing=function(){return x},this.dispose=function(){a.depthTexture&&a.depthTexture.dispose(),a.dispose(),o.dispose(),l.dispose(),c.dispose()}}const cp=new wn,Gc=new Is(1,1),up=new qd,fp=new I0,hp=new tp,yh=[],Eh=[],bh=new Float32Array(16),Th=new Float32Array(9),Ah=new Float32Array(4);function zs(i,e,t){const n=i[0];if(n<=0||n>0)return i;const r=e*t;let s=yh[r];if(s===void 0&&(s=new Float32Array(r),yh[r]=s),e!==0){n.toArray(s,0);for(let a=1,o=0;a!==e;++a)o+=t,i[a].toArray(s,o)}return s}function ln(i,e){if(i.length!==e.length)return!1;for(let t=0,n=i.length;t<n;t++)if(i[t]!==e[t])return!1;return!0}function cn(i,e){for(let t=0,n=e.length;t<n;t++)i[t]=e[t]}function Go(i,e){let t=Eh[e];t===void 0&&(t=new Int32Array(e),Eh[e]=t);for(let n=0;n!==e;++n)t[n]=i.allocateTextureUnit();return t}function xS(i,e){const t=this.cache;t[0]!==e&&(i.uniform1f(this.addr,e),t[0]=e)}function vS(i,e){const t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y)&&(i.uniform2f(this.addr,e.x,e.y),t[0]=e.x,t[1]=e.y);else{if(ln(t,e))return;i.uniform2fv(this.addr,e),cn(t,e)}}function SS(i,e){const t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z)&&(i.uniform3f(this.addr,e.x,e.y,e.z),t[0]=e.x,t[1]=e.y,t[2]=e.z);else if(e.r!==void 0)(t[0]!==e.r||t[1]!==e.g||t[2]!==e.b)&&(i.uniform3f(this.addr,e.r,e.g,e.b),t[0]=e.r,t[1]=e.g,t[2]=e.b);else{if(ln(t,e))return;i.uniform3fv(this.addr,e),cn(t,e)}}function MS(i,e){const t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z||t[3]!==e.w)&&(i.uniform4f(this.addr,e.x,e.y,e.z,e.w),t[0]=e.x,t[1]=e.y,t[2]=e.z,t[3]=e.w);else{if(ln(t,e))return;i.uniform4fv(this.addr,e),cn(t,e)}}function yS(i,e){const t=this.cache,n=e.elements;if(n===void 0){if(ln(t,e))return;i.uniformMatrix2fv(this.addr,!1,e),cn(t,e)}else{if(ln(t,n))return;Ah.set(n),i.uniformMatrix2fv(this.addr,!1,Ah),cn(t,n)}}function ES(i,e){const t=this.cache,n=e.elements;if(n===void 0){if(ln(t,e))return;i.uniformMatrix3fv(this.addr,!1,e),cn(t,e)}else{if(ln(t,n))return;Th.set(n),i.uniformMatrix3fv(this.addr,!1,Th),cn(t,n)}}function bS(i,e){const t=this.cache,n=e.elements;if(n===void 0){if(ln(t,e))return;i.uniformMatrix4fv(this.addr,!1,e),cn(t,e)}else{if(ln(t,n))return;bh.set(n),i.uniformMatrix4fv(this.addr,!1,bh),cn(t,n)}}function TS(i,e){const t=this.cache;t[0]!==e&&(i.uniform1i(this.addr,e),t[0]=e)}function AS(i,e){const t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y)&&(i.uniform2i(this.addr,e.x,e.y),t[0]=e.x,t[1]=e.y);else{if(ln(t,e))return;i.uniform2iv(this.addr,e),cn(t,e)}}function wS(i,e){const t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z)&&(i.uniform3i(this.addr,e.x,e.y,e.z),t[0]=e.x,t[1]=e.y,t[2]=e.z);else{if(ln(t,e))return;i.uniform3iv(this.addr,e),cn(t,e)}}function RS(i,e){const t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z||t[3]!==e.w)&&(i.uniform4i(this.addr,e.x,e.y,e.z,e.w),t[0]=e.x,t[1]=e.y,t[2]=e.z,t[3]=e.w);else{if(ln(t,e))return;i.uniform4iv(this.addr,e),cn(t,e)}}function CS(i,e){const t=this.cache;t[0]!==e&&(i.uniform1ui(this.addr,e),t[0]=e)}function PS(i,e){const t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y)&&(i.uniform2ui(this.addr,e.x,e.y),t[0]=e.x,t[1]=e.y);else{if(ln(t,e))return;i.uniform2uiv(this.addr,e),cn(t,e)}}function DS(i,e){const t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z)&&(i.uniform3ui(this.addr,e.x,e.y,e.z),t[0]=e.x,t[1]=e.y,t[2]=e.z);else{if(ln(t,e))return;i.uniform3uiv(this.addr,e),cn(t,e)}}function LS(i,e){const t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z||t[3]!==e.w)&&(i.uniform4ui(this.addr,e.x,e.y,e.z,e.w),t[0]=e.x,t[1]=e.y,t[2]=e.z,t[3]=e.w);else{if(ln(t,e))return;i.uniform4uiv(this.addr,e),cn(t,e)}}function IS(i,e,t){const n=this.cache,r=t.allocateTextureUnit();n[0]!==r&&(i.uniform1i(this.addr,r),n[0]=r);let s;this.type===i.SAMPLER_2D_SHADOW?(Gc.compareFunction=t.isReversedDepthBuffer()?hu:fu,s=Gc):s=cp,t.setTexture2D(e||s,r)}function US(i,e,t){const n=this.cache,r=t.allocateTextureUnit();n[0]!==r&&(i.uniform1i(this.addr,r),n[0]=r),t.setTexture3D(e||fp,r)}function NS(i,e,t){const n=this.cache,r=t.allocateTextureUnit();n[0]!==r&&(i.uniform1i(this.addr,r),n[0]=r),t.setTextureCube(e||hp,r)}function FS(i,e,t){const n=this.cache,r=t.allocateTextureUnit();n[0]!==r&&(i.uniform1i(this.addr,r),n[0]=r),t.setTexture2DArray(e||up,r)}function OS(i){switch(i){case 5126:return xS;case 35664:return vS;case 35665:return SS;case 35666:return MS;case 35674:return yS;case 35675:return ES;case 35676:return bS;case 5124:case 35670:return TS;case 35667:case 35671:return AS;case 35668:case 35672:return wS;case 35669:case 35673:return RS;case 5125:return CS;case 36294:return PS;case 36295:return DS;case 36296:return LS;case 35678:case 36198:case 36298:case 36306:case 35682:return IS;case 35679:case 36299:case 36307:return US;case 35680:case 36300:case 36308:case 36293:return NS;case 36289:case 36303:case 36311:case 36292:return FS}}function BS(i,e){i.uniform1fv(this.addr,e)}function zS(i,e){const t=zs(e,this.size,2);i.uniform2fv(this.addr,t)}function kS(i,e){const t=zs(e,this.size,3);i.uniform3fv(this.addr,t)}function VS(i,e){const t=zs(e,this.size,4);i.uniform4fv(this.addr,t)}function GS(i,e){const t=zs(e,this.size,4);i.uniformMatrix2fv(this.addr,!1,t)}function HS(i,e){const t=zs(e,this.size,9);i.uniformMatrix3fv(this.addr,!1,t)}function WS(i,e){const t=zs(e,this.size,16);i.uniformMatrix4fv(this.addr,!1,t)}function XS(i,e){i.uniform1iv(this.addr,e)}function YS(i,e){i.uniform2iv(this.addr,e)}function qS(i,e){i.uniform3iv(this.addr,e)}function KS(i,e){i.uniform4iv(this.addr,e)}function $S(i,e){i.uniform1uiv(this.addr,e)}function ZS(i,e){i.uniform2uiv(this.addr,e)}function JS(i,e){i.uniform3uiv(this.addr,e)}function jS(i,e){i.uniform4uiv(this.addr,e)}function QS(i,e,t){const n=this.cache,r=e.length,s=Go(t,r);ln(n,s)||(i.uniform1iv(this.addr,s),cn(n,s));let a;this.type===i.SAMPLER_2D_SHADOW?a=Gc:a=cp;for(let o=0;o!==r;++o)t.setTexture2D(e[o]||a,s[o])}function eM(i,e,t){const n=this.cache,r=e.length,s=Go(t,r);ln(n,s)||(i.uniform1iv(this.addr,s),cn(n,s));for(let a=0;a!==r;++a)t.setTexture3D(e[a]||fp,s[a])}function tM(i,e,t){const n=this.cache,r=e.length,s=Go(t,r);ln(n,s)||(i.uniform1iv(this.addr,s),cn(n,s));for(let a=0;a!==r;++a)t.setTextureCube(e[a]||hp,s[a])}function nM(i,e,t){const n=this.cache,r=e.length,s=Go(t,r);ln(n,s)||(i.uniform1iv(this.addr,s),cn(n,s));for(let a=0;a!==r;++a)t.setTexture2DArray(e[a]||up,s[a])}function iM(i){switch(i){case 5126:return BS;case 35664:return zS;case 35665:return kS;case 35666:return VS;case 35674:return GS;case 35675:return HS;case 35676:return WS;case 5124:case 35670:return XS;case 35667:case 35671:return YS;case 35668:case 35672:return qS;case 35669:case 35673:return KS;case 5125:return $S;case 36294:return ZS;case 36295:return JS;case 36296:return jS;case 35678:case 36198:case 36298:case 36306:case 35682:return QS;case 35679:case 36299:case 36307:return eM;case 35680:case 36300:case 36308:case 36293:return tM;case 36289:case 36303:case 36311:case 36292:return nM}}class rM{constructor(e,t,n){this.id=e,this.addr=n,this.cache=[],this.type=t.type,this.setValue=OS(t.type)}}class sM{constructor(e,t,n){this.id=e,this.addr=n,this.cache=[],this.type=t.type,this.size=t.size,this.setValue=iM(t.type)}}class aM{constructor(e){this.id=e,this.seq=[],this.map={}}setValue(e,t,n){const r=this.seq;for(let s=0,a=r.length;s!==a;++s){const o=r[s];o.setValue(e,t[o.id],n)}}}const Bl=/(\w+)(\])?(\[|\.)?/g;function wh(i,e){i.seq.push(e),i.map[e.id]=e}function oM(i,e,t){const n=i.name,r=n.length;for(Bl.lastIndex=0;;){const s=Bl.exec(n),a=Bl.lastIndex;let o=s[1];const l=s[2]==="]",c=s[3];if(l&&(o=o|0),c===void 0||c==="["&&a+2===r){wh(t,c===void 0?new rM(o,i,e):new sM(o,i,e));break}else{let h=t.map[o];h===void 0&&(h=new aM(o),wh(t,h)),t=h}}}class mo{constructor(e,t){this.seq=[],this.map={};const n=e.getProgramParameter(t,e.ACTIVE_UNIFORMS);for(let a=0;a<n;++a){const o=e.getActiveUniform(t,a),l=e.getUniformLocation(t,o.name);oM(o,l,this)}const r=[],s=[];for(const a of this.seq)a.type===e.SAMPLER_2D_SHADOW||a.type===e.SAMPLER_CUBE_SHADOW||a.type===e.SAMPLER_2D_ARRAY_SHADOW?r.push(a):s.push(a);r.length>0&&(this.seq=r.concat(s))}setValue(e,t,n,r){const s=this.map[t];s!==void 0&&s.setValue(e,n,r)}setOptional(e,t,n){const r=t[n];r!==void 0&&this.setValue(e,n,r)}static upload(e,t,n,r){for(let s=0,a=t.length;s!==a;++s){const o=t[s],l=n[o.id];l.needsUpdate!==!1&&o.setValue(e,l.value,r)}}static seqWithValue(e,t){const n=[];for(let r=0,s=e.length;r!==s;++r){const a=e[r];a.id in t&&n.push(a)}return n}}function Rh(i,e,t){const n=i.createShader(e);return i.shaderSource(n,t),i.compileShader(n),n}const lM=37297;let cM=0;function uM(i,e){const t=i.split(`
`),n=[],r=Math.max(e-6,0),s=Math.min(e+6,t.length);for(let a=r;a<s;a++){const o=a+1;n.push(`${o===e?">":" "} ${o}: ${t[a]}`)}return n.join(`
`)}const Ch=new nt;function fM(i){mt._getMatrix(Ch,mt.workingColorSpace,i);const e=`mat3( ${Ch.elements.map(t=>t.toFixed(4))} )`;switch(mt.getTransfer(i)){case yo:return[e,"LinearTransferOETF"];case Ct:return[e,"sRGBTransferOETF"];default:return Je("WebGLProgram: Unsupported color space: ",i),[e,"LinearTransferOETF"]}}function Ph(i,e,t){const n=i.getShaderParameter(e,i.COMPILE_STATUS),s=(i.getShaderInfoLog(e)||"").trim();if(n&&s==="")return"";const a=/ERROR: 0:(\d+)/.exec(s);if(a){const o=parseInt(a[1]);return t.toUpperCase()+`

`+s+`

`+uM(i.getShaderSource(e),o)}else return s}function hM(i,e){const t=fM(e);return[`vec4 ${i}( vec4 value ) {`,`	return ${t[1]}( vec4( value.rgb * ${t[0]}, value.a ) );`,"}"].join(`
`)}const dM={[Dd]:"Linear",[Ld]:"Reinhard",[Id]:"Cineon",[Ud]:"ACESFilmic",[Fd]:"AgX",[Od]:"Neutral",[Nd]:"Custom"};function pM(i,e){const t=dM[e];return t===void 0?(Je("WebGLProgram: Unsupported toneMapping:",e),"vec3 "+i+"( vec3 color ) { return LinearToneMapping( color ); }"):"vec3 "+i+"( vec3 color ) { return "+t+"ToneMapping( color ); }"}const io=new X;function mM(){mt.getLuminanceCoefficients(io);const i=io.x.toFixed(4),e=io.y.toFixed(4),t=io.z.toFixed(4);return["float luminance( const in vec3 rgb ) {",`	const vec3 weights = vec3( ${i}, ${e}, ${t} );`,"	return dot( weights, rgb );","}"].join(`
`)}function gM(i){return[i.extensionClipCullDistance?"#extension GL_ANGLE_clip_cull_distance : require":"",i.extensionMultiDraw?"#extension GL_ANGLE_multi_draw : require":""].filter(da).join(`
`)}function _M(i){const e=[];for(const t in i){const n=i[t];n!==!1&&e.push("#define "+t+" "+n)}return e.join(`
`)}function xM(i,e){const t={},n=i.getProgramParameter(e,i.ACTIVE_ATTRIBUTES);for(let r=0;r<n;r++){const s=i.getActiveAttrib(e,r),a=s.name;let o=1;s.type===i.FLOAT_MAT2&&(o=2),s.type===i.FLOAT_MAT3&&(o=3),s.type===i.FLOAT_MAT4&&(o=4),t[a]={type:s.type,location:i.getAttribLocation(e,a),locationSize:o}}return t}function da(i){return i!==""}function Dh(i,e){const t=e.numSpotLightShadows+e.numSpotLightMaps-e.numSpotLightShadowsWithMaps;return i.replace(/NUM_DIR_LIGHTS/g,e.numDirLights).replace(/NUM_SPOT_LIGHTS/g,e.numSpotLights).replace(/NUM_SPOT_LIGHT_MAPS/g,e.numSpotLightMaps).replace(/NUM_SPOT_LIGHT_COORDS/g,t).replace(/NUM_RECT_AREA_LIGHTS/g,e.numRectAreaLights).replace(/NUM_POINT_LIGHTS/g,e.numPointLights).replace(/NUM_HEMI_LIGHTS/g,e.numHemiLights).replace(/NUM_DIR_LIGHT_SHADOWS/g,e.numDirLightShadows).replace(/NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS/g,e.numSpotLightShadowsWithMaps).replace(/NUM_SPOT_LIGHT_SHADOWS/g,e.numSpotLightShadows).replace(/NUM_POINT_LIGHT_SHADOWS/g,e.numPointLightShadows)}function Lh(i,e){return i.replace(/NUM_CLIPPING_PLANES/g,e.numClippingPlanes).replace(/UNION_CLIPPING_PLANES/g,e.numClippingPlanes-e.numClipIntersection)}const vM=/^[ \t]*#include +<([\w\d./]+)>/gm;function Hc(i){return i.replace(vM,MM)}const SM=new Map;function MM(i,e){let t=st[e];if(t===void 0){const n=SM.get(e);if(n!==void 0)t=st[n],Je('WebGLRenderer: Shader chunk "%s" has been deprecated. Use "%s" instead.',e,n);else throw new Error("THREE.WebGLProgram: Can not resolve #include <"+e+">")}return Hc(t)}const yM=/#pragma unroll_loop_start\s+for\s*\(\s*int\s+i\s*=\s*(\d+)\s*;\s*i\s*<\s*(\d+)\s*;\s*i\s*\+\+\s*\)\s*{([\s\S]+?)}\s+#pragma unroll_loop_end/g;function Ih(i){return i.replace(yM,EM)}function EM(i,e,t,n){let r="";for(let s=parseInt(e);s<parseInt(t);s++)r+=n.replace(/\[\s*i\s*\]/g,"[ "+s+" ]").replace(/UNROLLED_LOOP_INDEX/g,s);return r}function Uh(i){let e=`precision ${i.precision} float;
	precision ${i.precision} int;
	precision ${i.precision} sampler2D;
	precision ${i.precision} samplerCube;
	precision ${i.precision} sampler3D;
	precision ${i.precision} sampler2DArray;
	precision ${i.precision} sampler2DShadow;
	precision ${i.precision} samplerCubeShadow;
	precision ${i.precision} sampler2DArrayShadow;
	precision ${i.precision} isampler2D;
	precision ${i.precision} isampler3D;
	precision ${i.precision} isamplerCube;
	precision ${i.precision} isampler2DArray;
	precision ${i.precision} usampler2D;
	precision ${i.precision} usampler3D;
	precision ${i.precision} usamplerCube;
	precision ${i.precision} usampler2DArray;
	`;return i.precision==="highp"?e+=`
#define HIGH_PRECISION`:i.precision==="mediump"?e+=`
#define MEDIUM_PRECISION`:i.precision==="lowp"&&(e+=`
#define LOW_PRECISION`),e}const bM={[oo]:"SHADOWMAP_TYPE_PCF",[fa]:"SHADOWMAP_TYPE_VSM"};function TM(i){return bM[i.shadowMapType]||"SHADOWMAP_TYPE_BASIC"}const AM={[kr]:"ENVMAP_TYPE_CUBE",[Ls]:"ENVMAP_TYPE_CUBE",[Oo]:"ENVMAP_TYPE_CUBE_UV"};function wM(i){return i.envMap===!1?"ENVMAP_TYPE_CUBE":AM[i.envMapMode]||"ENVMAP_TYPE_CUBE"}const RM={[Ls]:"ENVMAP_MODE_REFRACTION"};function CM(i){return i.envMap===!1?"ENVMAP_MODE_REFLECTION":RM[i.envMapMode]||"ENVMAP_MODE_REFLECTION"}const PM={[Pd]:"ENVMAP_BLENDING_MULTIPLY",[f0]:"ENVMAP_BLENDING_MIX",[h0]:"ENVMAP_BLENDING_ADD"};function DM(i){return i.envMap===!1?"ENVMAP_BLENDING_NONE":PM[i.combine]||"ENVMAP_BLENDING_NONE"}function LM(i){const e=i.envMapCubeUVHeight;if(e===null)return null;const t=Math.log2(e)-2,n=1/e;return{texelWidth:1/(3*Math.max(Math.pow(2,t),112)),texelHeight:n,maxMip:t}}function IM(i,e,t,n){const r=i.getContext(),s=t.defines;let a=t.vertexShader,o=t.fragmentShader;const l=TM(t),c=wM(t),f=CM(t),h=DM(t),u=LM(t),d=gM(t),x=_M(s),E=r.createProgram();let m,p,T=t.glslVersion?"#version "+t.glslVersion+`
`:"";t.isRawShaderMaterial?(m=["#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,x].filter(da).join(`
`),m.length>0&&(m+=`
`),p=["#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,x].filter(da).join(`
`),p.length>0&&(p+=`
`)):(m=[Uh(t),"#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,x,t.extensionClipCullDistance?"#define USE_CLIP_DISTANCE":"",t.batching?"#define USE_BATCHING":"",t.batchingColor?"#define USE_BATCHING_COLOR":"",t.instancing?"#define USE_INSTANCING":"",t.instancingColor?"#define USE_INSTANCING_COLOR":"",t.instancingMorph?"#define USE_INSTANCING_MORPH":"",t.useFog&&t.fog?"#define USE_FOG":"",t.useFog&&t.fogExp2?"#define FOG_EXP2":"",t.map?"#define USE_MAP":"",t.envMap?"#define USE_ENVMAP":"",t.envMap?"#define "+f:"",t.lightMap?"#define USE_LIGHTMAP":"",t.aoMap?"#define USE_AOMAP":"",t.bumpMap?"#define USE_BUMPMAP":"",t.normalMap?"#define USE_NORMALMAP":"",t.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",t.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",t.displacementMap?"#define USE_DISPLACEMENTMAP":"",t.emissiveMap?"#define USE_EMISSIVEMAP":"",t.anisotropy?"#define USE_ANISOTROPY":"",t.anisotropyMap?"#define USE_ANISOTROPYMAP":"",t.clearcoatMap?"#define USE_CLEARCOATMAP":"",t.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",t.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",t.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",t.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",t.specularMap?"#define USE_SPECULARMAP":"",t.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",t.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",t.roughnessMap?"#define USE_ROUGHNESSMAP":"",t.metalnessMap?"#define USE_METALNESSMAP":"",t.alphaMap?"#define USE_ALPHAMAP":"",t.alphaHash?"#define USE_ALPHAHASH":"",t.transmission?"#define USE_TRANSMISSION":"",t.transmissionMap?"#define USE_TRANSMISSIONMAP":"",t.thicknessMap?"#define USE_THICKNESSMAP":"",t.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",t.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",t.mapUv?"#define MAP_UV "+t.mapUv:"",t.alphaMapUv?"#define ALPHAMAP_UV "+t.alphaMapUv:"",t.lightMapUv?"#define LIGHTMAP_UV "+t.lightMapUv:"",t.aoMapUv?"#define AOMAP_UV "+t.aoMapUv:"",t.emissiveMapUv?"#define EMISSIVEMAP_UV "+t.emissiveMapUv:"",t.bumpMapUv?"#define BUMPMAP_UV "+t.bumpMapUv:"",t.normalMapUv?"#define NORMALMAP_UV "+t.normalMapUv:"",t.displacementMapUv?"#define DISPLACEMENTMAP_UV "+t.displacementMapUv:"",t.metalnessMapUv?"#define METALNESSMAP_UV "+t.metalnessMapUv:"",t.roughnessMapUv?"#define ROUGHNESSMAP_UV "+t.roughnessMapUv:"",t.anisotropyMapUv?"#define ANISOTROPYMAP_UV "+t.anisotropyMapUv:"",t.clearcoatMapUv?"#define CLEARCOATMAP_UV "+t.clearcoatMapUv:"",t.clearcoatNormalMapUv?"#define CLEARCOAT_NORMALMAP_UV "+t.clearcoatNormalMapUv:"",t.clearcoatRoughnessMapUv?"#define CLEARCOAT_ROUGHNESSMAP_UV "+t.clearcoatRoughnessMapUv:"",t.iridescenceMapUv?"#define IRIDESCENCEMAP_UV "+t.iridescenceMapUv:"",t.iridescenceThicknessMapUv?"#define IRIDESCENCE_THICKNESSMAP_UV "+t.iridescenceThicknessMapUv:"",t.sheenColorMapUv?"#define SHEEN_COLORMAP_UV "+t.sheenColorMapUv:"",t.sheenRoughnessMapUv?"#define SHEEN_ROUGHNESSMAP_UV "+t.sheenRoughnessMapUv:"",t.specularMapUv?"#define SPECULARMAP_UV "+t.specularMapUv:"",t.specularColorMapUv?"#define SPECULAR_COLORMAP_UV "+t.specularColorMapUv:"",t.specularIntensityMapUv?"#define SPECULAR_INTENSITYMAP_UV "+t.specularIntensityMapUv:"",t.transmissionMapUv?"#define TRANSMISSIONMAP_UV "+t.transmissionMapUv:"",t.thicknessMapUv?"#define THICKNESSMAP_UV "+t.thicknessMapUv:"",t.vertexTangents&&t.flatShading===!1?"#define USE_TANGENT":"",t.vertexNormals?"#define HAS_NORMAL":"",t.vertexColors?"#define USE_COLOR":"",t.vertexAlphas?"#define USE_COLOR_ALPHA":"",t.vertexUv1s?"#define USE_UV1":"",t.vertexUv2s?"#define USE_UV2":"",t.vertexUv3s?"#define USE_UV3":"",t.pointsUvs?"#define USE_POINTS_UV":"",t.flatShading?"#define FLAT_SHADED":"",t.skinning?"#define USE_SKINNING":"",t.morphTargets?"#define USE_MORPHTARGETS":"",t.morphNormals&&t.flatShading===!1?"#define USE_MORPHNORMALS":"",t.morphColors?"#define USE_MORPHCOLORS":"",t.morphTargetsCount>0?"#define MORPHTARGETS_TEXTURE_STRIDE "+t.morphTextureStride:"",t.morphTargetsCount>0?"#define MORPHTARGETS_COUNT "+t.morphTargetsCount:"",t.doubleSided?"#define DOUBLE_SIDED":"",t.flipSided?"#define FLIP_SIDED":"",t.shadowMapEnabled?"#define USE_SHADOWMAP":"",t.shadowMapEnabled?"#define "+l:"",t.sizeAttenuation?"#define USE_SIZEATTENUATION":"",t.numLightProbes>0?"#define USE_LIGHT_PROBES":"",t.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",t.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 modelMatrix;","uniform mat4 modelViewMatrix;","uniform mat4 projectionMatrix;","uniform mat4 viewMatrix;","uniform mat3 normalMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;","#ifdef USE_INSTANCING","	attribute mat4 instanceMatrix;","#endif","#ifdef USE_INSTANCING_COLOR","	attribute vec3 instanceColor;","#endif","#ifdef USE_INSTANCING_MORPH","	uniform sampler2D morphTexture;","#endif","attribute vec3 position;","attribute vec3 normal;","attribute vec2 uv;","#ifdef USE_UV1","	attribute vec2 uv1;","#endif","#ifdef USE_UV2","	attribute vec2 uv2;","#endif","#ifdef USE_UV3","	attribute vec2 uv3;","#endif","#ifdef USE_TANGENT","	attribute vec4 tangent;","#endif","#if defined( USE_COLOR_ALPHA )","	attribute vec4 color;","#elif defined( USE_COLOR )","	attribute vec3 color;","#endif","#ifdef USE_SKINNING","	attribute vec4 skinIndex;","	attribute vec4 skinWeight;","#endif",`
`].filter(da).join(`
`),p=[Uh(t),"#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,x,t.useFog&&t.fog?"#define USE_FOG":"",t.useFog&&t.fogExp2?"#define FOG_EXP2":"",t.alphaToCoverage?"#define ALPHA_TO_COVERAGE":"",t.map?"#define USE_MAP":"",t.matcap?"#define USE_MATCAP":"",t.envMap?"#define USE_ENVMAP":"",t.envMap?"#define "+c:"",t.envMap?"#define "+f:"",t.envMap?"#define "+h:"",u?"#define CUBEUV_TEXEL_WIDTH "+u.texelWidth:"",u?"#define CUBEUV_TEXEL_HEIGHT "+u.texelHeight:"",u?"#define CUBEUV_MAX_MIP "+u.maxMip+".0":"",t.lightMap?"#define USE_LIGHTMAP":"",t.aoMap?"#define USE_AOMAP":"",t.bumpMap?"#define USE_BUMPMAP":"",t.normalMap?"#define USE_NORMALMAP":"",t.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",t.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",t.packedNormalMap?"#define USE_PACKED_NORMALMAP":"",t.emissiveMap?"#define USE_EMISSIVEMAP":"",t.anisotropy?"#define USE_ANISOTROPY":"",t.anisotropyMap?"#define USE_ANISOTROPYMAP":"",t.clearcoat?"#define USE_CLEARCOAT":"",t.clearcoatMap?"#define USE_CLEARCOATMAP":"",t.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",t.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",t.dispersion?"#define USE_DISPERSION":"",t.iridescence?"#define USE_IRIDESCENCE":"",t.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",t.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",t.specularMap?"#define USE_SPECULARMAP":"",t.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",t.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",t.roughnessMap?"#define USE_ROUGHNESSMAP":"",t.metalnessMap?"#define USE_METALNESSMAP":"",t.alphaMap?"#define USE_ALPHAMAP":"",t.alphaTest?"#define USE_ALPHATEST":"",t.alphaHash?"#define USE_ALPHAHASH":"",t.sheen?"#define USE_SHEEN":"",t.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",t.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",t.transmission?"#define USE_TRANSMISSION":"",t.transmissionMap?"#define USE_TRANSMISSIONMAP":"",t.thicknessMap?"#define USE_THICKNESSMAP":"",t.vertexTangents&&t.flatShading===!1?"#define USE_TANGENT":"",t.vertexColors||t.instancingColor?"#define USE_COLOR":"",t.vertexAlphas||t.batchingColor?"#define USE_COLOR_ALPHA":"",t.vertexUv1s?"#define USE_UV1":"",t.vertexUv2s?"#define USE_UV2":"",t.vertexUv3s?"#define USE_UV3":"",t.pointsUvs?"#define USE_POINTS_UV":"",t.gradientMap?"#define USE_GRADIENTMAP":"",t.flatShading?"#define FLAT_SHADED":"",t.doubleSided?"#define DOUBLE_SIDED":"",t.flipSided?"#define FLIP_SIDED":"",t.shadowMapEnabled?"#define USE_SHADOWMAP":"",t.shadowMapEnabled?"#define "+l:"",t.premultipliedAlpha?"#define PREMULTIPLIED_ALPHA":"",t.numLightProbes>0?"#define USE_LIGHT_PROBES":"",t.numLightProbeGrids>0?"#define USE_LIGHT_PROBES_GRID":"",t.decodeVideoTexture?"#define DECODE_VIDEO_TEXTURE":"",t.decodeVideoTextureEmissive?"#define DECODE_VIDEO_TEXTURE_EMISSIVE":"",t.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",t.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 viewMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;",t.toneMapping!==Ui?"#define TONE_MAPPING":"",t.toneMapping!==Ui?st.tonemapping_pars_fragment:"",t.toneMapping!==Ui?pM("toneMapping",t.toneMapping):"",t.dithering?"#define DITHERING":"",t.opaque?"#define OPAQUE":"",st.colorspace_pars_fragment,hM("linearToOutputTexel",t.outputColorSpace),mM(),t.useDepthPacking?"#define DEPTH_PACKING "+t.depthPacking:"",`
`].filter(da).join(`
`)),a=Hc(a),a=Dh(a,t),a=Lh(a,t),o=Hc(o),o=Dh(o,t),o=Lh(o,t),a=Ih(a),o=Ih(o),t.isRawShaderMaterial!==!0&&(T=`#version 300 es
`,m=[d,"#define attribute in","#define varying out","#define texture2D texture"].join(`
`)+`
`+m,p=["#define varying in",t.glslVersion===zf?"":"layout(location = 0) out highp vec4 pc_fragColor;",t.glslVersion===zf?"":"#define gl_FragColor pc_fragColor","#define gl_FragDepthEXT gl_FragDepth","#define texture2D texture","#define textureCube texture","#define texture2DProj textureProj","#define texture2DLodEXT textureLod","#define texture2DProjLodEXT textureProjLod","#define textureCubeLodEXT textureLod","#define texture2DGradEXT textureGrad","#define texture2DProjGradEXT textureProjGrad","#define textureCubeGradEXT textureGrad"].join(`
`)+`
`+p);const P=T+m+a,S=T+p+o,C=Rh(r,r.VERTEX_SHADER,P),y=Rh(r,r.FRAGMENT_SHADER,S);r.attachShader(E,C),r.attachShader(E,y),t.index0AttributeName!==void 0?r.bindAttribLocation(E,0,t.index0AttributeName):t.hasPositionAttribute===!0&&r.bindAttribLocation(E,0,"position"),r.linkProgram(E);function I(D){if(i.debug.checkShaderErrors){const z=r.getProgramInfoLog(E)||"",O=r.getShaderInfoLog(C)||"",k=r.getShaderInfoLog(y)||"",L=z.trim(),N=O.trim(),B=k.trim();let j=!0,ne=!0;if(r.getProgramParameter(E,r.LINK_STATUS)===!1)if(j=!1,typeof i.debug.onShaderError=="function")i.debug.onShaderError(r,E,C,y);else{const ae=Ph(r,C,"vertex"),fe=Ph(r,y,"fragment");gt("WebGLProgram: Shader Error "+r.getError()+" - VALIDATE_STATUS "+r.getProgramParameter(E,r.VALIDATE_STATUS)+`

Material Name: `+D.name+`
Material Type: `+D.type+`

Program Info Log: `+L+`
`+ae+`
`+fe)}else L!==""?Je("WebGLProgram: Program Info Log:",L):(N===""||B==="")&&(ne=!1);ne&&(D.diagnostics={runnable:j,programLog:L,vertexShader:{log:N,prefix:m},fragmentShader:{log:B,prefix:p}})}r.deleteShader(C),r.deleteShader(y),v=new mo(r,E),A=xM(r,E)}let v;this.getUniforms=function(){return v===void 0&&I(this),v};let A;this.getAttributes=function(){return A===void 0&&I(this),A};let F=t.rendererExtensionParallelShaderCompile===!1;return this.isReady=function(){return F===!1&&(F=r.getProgramParameter(E,lM)),F},this.destroy=function(){n.releaseStatesOfProgram(this),r.deleteProgram(E),this.program=void 0},this.type=t.shaderType,this.name=t.shaderName,this.id=cM++,this.cacheKey=e,this.usedTimes=1,this.program=E,this.vertexShader=C,this.fragmentShader=y,this}let UM=0;class NM{constructor(){this.shaderCache=new Map,this.materialCache=new Map}update(e,t,n){const r=this._getShaderCacheForMaterial(e);return r.has(t)===!1&&(r.add(t),t.usedTimes++),r.has(n)===!1&&(r.add(n),n.usedTimes++),this}remove(e){const t=this.materialCache.get(e);for(const n of t)n.usedTimes--,n.usedTimes===0&&this.shaderCache.delete(n.code);return this.materialCache.delete(e),this}getVertexShaderStage(e){return this._getShaderStage(e.vertexShader)}getFragmentShaderStage(e){return this._getShaderStage(e.fragmentShader)}dispose(){this.shaderCache.clear(),this.materialCache.clear()}_getShaderCacheForMaterial(e){const t=this.materialCache;let n=t.get(e);return n===void 0&&(n=new Set,t.set(e,n)),n}_getShaderStage(e){const t=this.shaderCache;let n=t.get(e);return n===void 0&&(n=new FM(e),t.set(e,n)),n}}class FM{constructor(e){this.id=UM++,this.code=e,this.usedTimes=0}}function OM(i){return i===Vr||i===vo||i===So}function BM(i,e,t,n,r,s){const a=new Kd,o=new NM,l=new Set,c=[],f=new Map,h=n.logarithmicDepthBuffer;let u=n.precision;const d={MeshDepthMaterial:"depth",MeshDistanceMaterial:"distance",MeshNormalMaterial:"normal",MeshBasicMaterial:"basic",MeshLambertMaterial:"lambert",MeshPhongMaterial:"phong",MeshToonMaterial:"toon",MeshStandardMaterial:"physical",MeshPhysicalMaterial:"physical",MeshMatcapMaterial:"matcap",LineBasicMaterial:"basic",LineDashedMaterial:"dashed",PointsMaterial:"points",ShadowMaterial:"shadow",SpriteMaterial:"sprite"};function x(v){return l.add(v),v===0?"uv":`uv${v}`}function E(v,A,F,D,z,O){const k=D.fog,L=z.geometry,N=v.isMeshStandardMaterial||v.isMeshLambertMaterial||v.isMeshPhongMaterial?D.environment:null,B=v.isMeshStandardMaterial||v.isMeshLambertMaterial&&!v.envMap||v.isMeshPhongMaterial&&!v.envMap,j=e.get(v.envMap||N,B),ne=j&&j.mapping===Oo?j.image.height:null,ae=d[v.type];v.precision!==null&&(u=n.getMaxPrecision(v.precision),u!==v.precision&&Je("WebGLProgram.getParameters:",v.precision,"not supported, using",u,"instead."));const fe=L.morphAttributes.position||L.morphAttributes.normal||L.morphAttributes.color,re=fe!==void 0?fe.length:0;let le=0;L.morphAttributes.position!==void 0&&(le=1),L.morphAttributes.normal!==void 0&&(le=2),L.morphAttributes.color!==void 0&&(le=3);let Ze,Ke,Q,ue;if(ae){const Ne=Ri[ae];Ze=Ne.vertexShader,Ke=Ne.fragmentShader}else{Ze=v.vertexShader,Ke=v.fragmentShader;const Ne=o.getVertexShaderStage(v),at=o.getFragmentShaderStage(v);o.update(v,Ne,at),Q=Ne.id,ue=at.id}const he=i.getRenderTarget(),Be=i.state.buffers.depth.getReversed(),Ue=z.isInstancedMesh===!0,ze=z.isBatchedMesh===!0,St=!!v.map,je=!!v.matcap,ht=!!j,ot=!!v.aoMap,rt=!!v.lightMap,Ft=!!v.bumpMap&&v.wireframe===!1,Gt=!!v.normalMap,Ot=!!v.displacementMap,_t=!!v.emissiveMap,wt=!!v.metalnessMap,Tt=!!v.roughnessMap,H=v.anisotropy>0,Xe=v.clearcoat>0,pe=v.dispersion>0,w=v.iridescence>0,_=v.sheen>0,Y=v.transmission>0,$=H&&!!v.anisotropyMap,ee=Xe&&!!v.clearcoatMap,ge=Xe&&!!v.clearcoatNormalMap,_e=Xe&&!!v.clearcoatRoughnessMap,te=w&&!!v.iridescenceMap,ie=w&&!!v.iridescenceThicknessMap,xe=_&&!!v.sheenColorMap,Ve=_&&!!v.sheenRoughnessMap,ye=!!v.specularMap,Se=!!v.specularColorMap,ke=!!v.specularIntensityMap,Ye=Y&&!!v.transmissionMap,qe=Y&&!!v.thicknessMap,G=!!v.gradientMap,Me=!!v.alphaMap,oe=v.alphaTest>0,Ee=!!v.alphaHash,Re=!!v.extensions;let de=Ui;v.toneMapped&&(he===null||he.isXRRenderTarget===!0)&&(de=i.toneMapping);const Ge={shaderID:ae,shaderType:v.type,shaderName:v.name,vertexShader:Ze,fragmentShader:Ke,defines:v.defines,customVertexShaderID:Q,customFragmentShaderID:ue,isRawShaderMaterial:v.isRawShaderMaterial===!0,glslVersion:v.glslVersion,precision:u,batching:ze,batchingColor:ze&&z._colorsTexture!==null,instancing:Ue,instancingColor:Ue&&z.instanceColor!==null,instancingMorph:Ue&&z.morphTexture!==null,outputColorSpace:he===null?i.outputColorSpace:he.isXRRenderTarget===!0?he.texture.colorSpace:mt.workingColorSpace,alphaToCoverage:!!v.alphaToCoverage,map:St,matcap:je,envMap:ht,envMapMode:ht&&j.mapping,envMapCubeUVHeight:ne,aoMap:ot,lightMap:rt,bumpMap:Ft,normalMap:Gt,displacementMap:Ot,emissiveMap:_t,normalMapObjectSpace:Gt&&v.normalMapType===m0,normalMapTangentSpace:Gt&&v.normalMapType===Of,packedNormalMap:Gt&&v.normalMapType===Of&&OM(v.normalMap.format),metalnessMap:wt,roughnessMap:Tt,anisotropy:H,anisotropyMap:$,clearcoat:Xe,clearcoatMap:ee,clearcoatNormalMap:ge,clearcoatRoughnessMap:_e,dispersion:pe,iridescence:w,iridescenceMap:te,iridescenceThicknessMap:ie,sheen:_,sheenColorMap:xe,sheenRoughnessMap:Ve,specularMap:ye,specularColorMap:Se,specularIntensityMap:ke,transmission:Y,transmissionMap:Ye,thicknessMap:qe,gradientMap:G,opaque:v.transparent===!1&&v.blending===bs&&v.alphaToCoverage===!1,alphaMap:Me,alphaTest:oe,alphaHash:Ee,combine:v.combine,mapUv:St&&x(v.map.channel),aoMapUv:ot&&x(v.aoMap.channel),lightMapUv:rt&&x(v.lightMap.channel),bumpMapUv:Ft&&x(v.bumpMap.channel),normalMapUv:Gt&&x(v.normalMap.channel),displacementMapUv:Ot&&x(v.displacementMap.channel),emissiveMapUv:_t&&x(v.emissiveMap.channel),metalnessMapUv:wt&&x(v.metalnessMap.channel),roughnessMapUv:Tt&&x(v.roughnessMap.channel),anisotropyMapUv:$&&x(v.anisotropyMap.channel),clearcoatMapUv:ee&&x(v.clearcoatMap.channel),clearcoatNormalMapUv:ge&&x(v.clearcoatNormalMap.channel),clearcoatRoughnessMapUv:_e&&x(v.clearcoatRoughnessMap.channel),iridescenceMapUv:te&&x(v.iridescenceMap.channel),iridescenceThicknessMapUv:ie&&x(v.iridescenceThicknessMap.channel),sheenColorMapUv:xe&&x(v.sheenColorMap.channel),sheenRoughnessMapUv:Ve&&x(v.sheenRoughnessMap.channel),specularMapUv:ye&&x(v.specularMap.channel),specularColorMapUv:Se&&x(v.specularColorMap.channel),specularIntensityMapUv:ke&&x(v.specularIntensityMap.channel),transmissionMapUv:Ye&&x(v.transmissionMap.channel),thicknessMapUv:qe&&x(v.thicknessMap.channel),alphaMapUv:Me&&x(v.alphaMap.channel),vertexTangents:!!L.attributes.tangent&&(Gt||H),vertexNormals:!!L.attributes.normal,vertexColors:v.vertexColors,vertexAlphas:v.vertexColors===!0&&!!L.attributes.color&&L.attributes.color.itemSize===4,pointsUvs:z.isPoints===!0&&!!L.attributes.uv&&(St||Me),fog:!!k,useFog:v.fog===!0,fogExp2:!!k&&k.isFogExp2,flatShading:v.wireframe===!1&&(v.flatShading===!0||L.attributes.normal===void 0&&Gt===!1&&(v.isMeshLambertMaterial||v.isMeshPhongMaterial||v.isMeshStandardMaterial||v.isMeshPhysicalMaterial)),sizeAttenuation:v.sizeAttenuation===!0,logarithmicDepthBuffer:h,reversedDepthBuffer:Be,skinning:z.isSkinnedMesh===!0,hasPositionAttribute:L.attributes.position!==void 0,morphTargets:L.morphAttributes.position!==void 0,morphNormals:L.morphAttributes.normal!==void 0,morphColors:L.morphAttributes.color!==void 0,morphTargetsCount:re,morphTextureStride:le,numDirLights:A.directional.length,numPointLights:A.point.length,numSpotLights:A.spot.length,numSpotLightMaps:A.spotLightMap.length,numRectAreaLights:A.rectArea.length,numHemiLights:A.hemi.length,numDirLightShadows:A.directionalShadowMap.length,numPointLightShadows:A.pointShadowMap.length,numSpotLightShadows:A.spotShadowMap.length,numSpotLightShadowsWithMaps:A.numSpotLightShadowsWithMaps,numLightProbes:A.numLightProbes,numLightProbeGrids:O.length,numClippingPlanes:s.numPlanes,numClipIntersection:s.numIntersection,dithering:v.dithering,shadowMapEnabled:i.shadowMap.enabled&&F.length>0,shadowMapType:i.shadowMap.type,toneMapping:de,decodeVideoTexture:St&&v.map.isVideoTexture===!0&&mt.getTransfer(v.map.colorSpace)===Ct,decodeVideoTextureEmissive:_t&&v.emissiveMap.isVideoTexture===!0&&mt.getTransfer(v.emissiveMap.colorSpace)===Ct,premultipliedAlpha:v.premultipliedAlpha,doubleSided:v.side===Pi,flipSided:v.side===Vn,useDepthPacking:v.depthPacking>=0,depthPacking:v.depthPacking||0,index0AttributeName:v.index0AttributeName,extensionClipCullDistance:Re&&v.extensions.clipCullDistance===!0&&t.has("WEBGL_clip_cull_distance"),extensionMultiDraw:(Re&&v.extensions.multiDraw===!0||ze)&&t.has("WEBGL_multi_draw"),rendererExtensionParallelShaderCompile:t.has("KHR_parallel_shader_compile"),customProgramCacheKey:v.customProgramCacheKey()};return Ge.vertexUv1s=l.has(1),Ge.vertexUv2s=l.has(2),Ge.vertexUv3s=l.has(3),l.clear(),Ge}function m(v){const A=[];if(v.shaderID?A.push(v.shaderID):(A.push(v.customVertexShaderID),A.push(v.customFragmentShaderID)),v.defines!==void 0)for(const F in v.defines)A.push(F),A.push(v.defines[F]);return v.isRawShaderMaterial===!1&&(p(A,v),T(A,v),A.push(i.outputColorSpace)),A.push(v.customProgramCacheKey),A.join()}function p(v,A){v.push(A.precision),v.push(A.outputColorSpace),v.push(A.envMapMode),v.push(A.envMapCubeUVHeight),v.push(A.mapUv),v.push(A.alphaMapUv),v.push(A.lightMapUv),v.push(A.aoMapUv),v.push(A.bumpMapUv),v.push(A.normalMapUv),v.push(A.displacementMapUv),v.push(A.emissiveMapUv),v.push(A.metalnessMapUv),v.push(A.roughnessMapUv),v.push(A.anisotropyMapUv),v.push(A.clearcoatMapUv),v.push(A.clearcoatNormalMapUv),v.push(A.clearcoatRoughnessMapUv),v.push(A.iridescenceMapUv),v.push(A.iridescenceThicknessMapUv),v.push(A.sheenColorMapUv),v.push(A.sheenRoughnessMapUv),v.push(A.specularMapUv),v.push(A.specularColorMapUv),v.push(A.specularIntensityMapUv),v.push(A.transmissionMapUv),v.push(A.thicknessMapUv),v.push(A.combine),v.push(A.fogExp2),v.push(A.sizeAttenuation),v.push(A.morphTargetsCount),v.push(A.morphAttributeCount),v.push(A.numDirLights),v.push(A.numPointLights),v.push(A.numSpotLights),v.push(A.numSpotLightMaps),v.push(A.numHemiLights),v.push(A.numRectAreaLights),v.push(A.numDirLightShadows),v.push(A.numPointLightShadows),v.push(A.numSpotLightShadows),v.push(A.numSpotLightShadowsWithMaps),v.push(A.numLightProbes),v.push(A.shadowMapType),v.push(A.toneMapping),v.push(A.numClippingPlanes),v.push(A.numClipIntersection),v.push(A.depthPacking)}function T(v,A){a.disableAll(),A.instancing&&a.enable(0),A.instancingColor&&a.enable(1),A.instancingMorph&&a.enable(2),A.matcap&&a.enable(3),A.envMap&&a.enable(4),A.normalMapObjectSpace&&a.enable(5),A.normalMapTangentSpace&&a.enable(6),A.clearcoat&&a.enable(7),A.iridescence&&a.enable(8),A.alphaTest&&a.enable(9),A.vertexColors&&a.enable(10),A.vertexAlphas&&a.enable(11),A.vertexUv1s&&a.enable(12),A.vertexUv2s&&a.enable(13),A.vertexUv3s&&a.enable(14),A.vertexTangents&&a.enable(15),A.anisotropy&&a.enable(16),A.alphaHash&&a.enable(17),A.batching&&a.enable(18),A.dispersion&&a.enable(19),A.batchingColor&&a.enable(20),A.gradientMap&&a.enable(21),A.packedNormalMap&&a.enable(22),A.vertexNormals&&a.enable(23),v.push(a.mask),a.disableAll(),A.fog&&a.enable(0),A.useFog&&a.enable(1),A.flatShading&&a.enable(2),A.logarithmicDepthBuffer&&a.enable(3),A.reversedDepthBuffer&&a.enable(4),A.skinning&&a.enable(5),A.morphTargets&&a.enable(6),A.morphNormals&&a.enable(7),A.morphColors&&a.enable(8),A.premultipliedAlpha&&a.enable(9),A.shadowMapEnabled&&a.enable(10),A.doubleSided&&a.enable(11),A.flipSided&&a.enable(12),A.useDepthPacking&&a.enable(13),A.dithering&&a.enable(14),A.transmission&&a.enable(15),A.sheen&&a.enable(16),A.opaque&&a.enable(17),A.pointsUvs&&a.enable(18),A.decodeVideoTexture&&a.enable(19),A.decodeVideoTextureEmissive&&a.enable(20),A.alphaToCoverage&&a.enable(21),A.numLightProbeGrids>0&&a.enable(22),A.hasPositionAttribute&&a.enable(23),v.push(a.mask)}function P(v){const A=d[v.type];let F;if(A){const D=Ri[A];F=n_.clone(D.uniforms)}else F=v.uniforms;return F}function S(v,A){let F=f.get(A);return F!==void 0?++F.usedTimes:(F=new IM(i,A,v,r),c.push(F),f.set(A,F)),F}function C(v){if(--v.usedTimes===0){const A=c.indexOf(v);c[A]=c[c.length-1],c.pop(),f.delete(v.cacheKey),v.destroy()}}function y(v){o.remove(v)}function I(){o.dispose()}return{getParameters:E,getProgramCacheKey:m,getUniforms:P,acquireProgram:S,releaseProgram:C,releaseShaderCache:y,programs:c,dispose:I}}function zM(){let i=new WeakMap;function e(a){return i.has(a)}function t(a){let o=i.get(a);return o===void 0&&(o={},i.set(a,o)),o}function n(a){i.delete(a)}function r(a,o,l){i.get(a)[o]=l}function s(){i=new WeakMap}return{has:e,get:t,remove:n,update:r,dispose:s}}function kM(i,e){return i.groupOrder!==e.groupOrder?i.groupOrder-e.groupOrder:i.renderOrder!==e.renderOrder?i.renderOrder-e.renderOrder:i.material.id!==e.material.id?i.material.id-e.material.id:i.materialVariant!==e.materialVariant?i.materialVariant-e.materialVariant:i.z!==e.z?i.z-e.z:i.id-e.id}function Nh(i,e){return i.groupOrder!==e.groupOrder?i.groupOrder-e.groupOrder:i.renderOrder!==e.renderOrder?i.renderOrder-e.renderOrder:i.z!==e.z?e.z-i.z:i.id-e.id}function Fh(){const i=[];let e=0;const t=[],n=[],r=[];function s(){e=0,t.length=0,n.length=0,r.length=0}function a(u){let d=0;return u.isInstancedMesh&&(d+=2),u.isSkinnedMesh&&(d+=1),d}function o(u,d,x,E,m,p){let T=i[e];return T===void 0?(T={id:u.id,object:u,geometry:d,material:x,materialVariant:a(u),groupOrder:E,renderOrder:u.renderOrder,z:m,group:p},i[e]=T):(T.id=u.id,T.object=u,T.geometry=d,T.material=x,T.materialVariant=a(u),T.groupOrder=E,T.renderOrder=u.renderOrder,T.z=m,T.group=p),e++,T}function l(u,d,x,E,m,p){const T=o(u,d,x,E,m,p);x.transmission>0?n.push(T):x.transparent===!0?r.push(T):t.push(T)}function c(u,d,x,E,m,p){const T=o(u,d,x,E,m,p);x.transmission>0?n.unshift(T):x.transparent===!0?r.unshift(T):t.unshift(T)}function f(u,d,x){t.length>1&&t.sort(u||kM),n.length>1&&n.sort(d||Nh),r.length>1&&r.sort(d||Nh),x&&(t.reverse(),n.reverse(),r.reverse())}function h(){for(let u=e,d=i.length;u<d;u++){const x=i[u];if(x.id===null)break;x.id=null,x.object=null,x.geometry=null,x.material=null,x.group=null}}return{opaque:t,transmissive:n,transparent:r,init:s,push:l,unshift:c,finish:h,sort:f}}function VM(){let i=new WeakMap;function e(n,r){const s=i.get(n);let a;return s===void 0?(a=new Fh,i.set(n,[a])):r>=s.length?(a=new Fh,s.push(a)):a=s[r],a}function t(){i=new WeakMap}return{get:e,dispose:t}}function GM(){const i={};return{get:function(e){if(i[e.id]!==void 0)return i[e.id];let t;switch(e.type){case"DirectionalLight":t={direction:new X,color:new ft};break;case"SpotLight":t={position:new X,direction:new X,color:new ft,distance:0,coneCos:0,penumbraCos:0,decay:0};break;case"PointLight":t={position:new X,color:new ft,distance:0,decay:0};break;case"HemisphereLight":t={direction:new X,skyColor:new ft,groundColor:new ft};break;case"RectAreaLight":t={color:new ft,position:new X,halfWidth:new X,halfHeight:new X};break}return i[e.id]=t,t}}}function HM(){const i={};return{get:function(e){if(i[e.id]!==void 0)return i[e.id];let t;switch(e.type){case"DirectionalLight":t={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new $e};break;case"SpotLight":t={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new $e};break;case"PointLight":t={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new $e,shadowCameraNear:1,shadowCameraFar:1e3};break}return i[e.id]=t,t}}}let WM=0;function XM(i,e){return(e.castShadow?2:0)-(i.castShadow?2:0)+(e.map?1:0)-(i.map?1:0)}function YM(i){const e=new GM,t=HM(),n={version:0,hash:{directionalLength:-1,pointLength:-1,spotLength:-1,rectAreaLength:-1,hemiLength:-1,numDirectionalShadows:-1,numPointShadows:-1,numSpotShadows:-1,numSpotMaps:-1,numLightProbes:-1},ambient:[0,0,0],probe:[],directional:[],directionalShadow:[],directionalShadowMap:[],directionalShadowMatrix:[],spot:[],spotLightMap:[],spotShadow:[],spotShadowMap:[],spotLightMatrix:[],rectArea:[],rectAreaLTC1:null,rectAreaLTC2:null,point:[],pointShadow:[],pointShadowMap:[],pointShadowMatrix:[],hemi:[],numSpotLightShadowsWithMaps:0,numLightProbes:0};for(let c=0;c<9;c++)n.probe.push(new X);const r=new X,s=new Xt,a=new Xt;function o(c){let f=0,h=0,u=0;for(let A=0;A<9;A++)n.probe[A].set(0,0,0);let d=0,x=0,E=0,m=0,p=0,T=0,P=0,S=0,C=0,y=0,I=0;c.sort(XM);for(let A=0,F=c.length;A<F;A++){const D=c[A],z=D.color,O=D.intensity,k=D.distance;let L=null;if(D.shadow&&D.shadow.map&&(D.shadow.map.texture.format===Vr?L=D.shadow.map.texture:L=D.shadow.map.depthTexture||D.shadow.map.texture),D.isAmbientLight)f+=z.r*O,h+=z.g*O,u+=z.b*O;else if(D.isLightProbe){for(let N=0;N<9;N++)n.probe[N].addScaledVector(D.sh.coefficients[N],O);I++}else if(D.isDirectionalLight){const N=e.get(D);if(N.color.copy(D.color).multiplyScalar(D.intensity),D.castShadow){const B=D.shadow,j=t.get(D);j.shadowIntensity=B.intensity,j.shadowBias=B.bias,j.shadowNormalBias=B.normalBias,j.shadowRadius=B.radius,j.shadowMapSize=B.mapSize,n.directionalShadow[d]=j,n.directionalShadowMap[d]=L,n.directionalShadowMatrix[d]=D.shadow.matrix,T++}n.directional[d]=N,d++}else if(D.isSpotLight){const N=e.get(D);N.position.setFromMatrixPosition(D.matrixWorld),N.color.copy(z).multiplyScalar(O),N.distance=k,N.coneCos=Math.cos(D.angle),N.penumbraCos=Math.cos(D.angle*(1-D.penumbra)),N.decay=D.decay,n.spot[E]=N;const B=D.shadow;if(D.map&&(n.spotLightMap[C]=D.map,C++,B.updateMatrices(D),D.castShadow&&y++),n.spotLightMatrix[E]=B.matrix,D.castShadow){const j=t.get(D);j.shadowIntensity=B.intensity,j.shadowBias=B.bias,j.shadowNormalBias=B.normalBias,j.shadowRadius=B.radius,j.shadowMapSize=B.mapSize,n.spotShadow[E]=j,n.spotShadowMap[E]=L,S++}E++}else if(D.isRectAreaLight){const N=e.get(D);N.color.copy(z).multiplyScalar(O),N.halfWidth.set(D.width*.5,0,0),N.halfHeight.set(0,D.height*.5,0),n.rectArea[m]=N,m++}else if(D.isPointLight){const N=e.get(D);if(N.color.copy(D.color).multiplyScalar(D.intensity),N.distance=D.distance,N.decay=D.decay,D.castShadow){const B=D.shadow,j=t.get(D);j.shadowIntensity=B.intensity,j.shadowBias=B.bias,j.shadowNormalBias=B.normalBias,j.shadowRadius=B.radius,j.shadowMapSize=B.mapSize,j.shadowCameraNear=B.camera.near,j.shadowCameraFar=B.camera.far,n.pointShadow[x]=j,n.pointShadowMap[x]=L,n.pointShadowMatrix[x]=D.shadow.matrix,P++}n.point[x]=N,x++}else if(D.isHemisphereLight){const N=e.get(D);N.skyColor.copy(D.color).multiplyScalar(O),N.groundColor.copy(D.groundColor).multiplyScalar(O),n.hemi[p]=N,p++}}m>0&&(i.has("OES_texture_float_linear")===!0?(n.rectAreaLTC1=we.LTC_FLOAT_1,n.rectAreaLTC2=we.LTC_FLOAT_2):(n.rectAreaLTC1=we.LTC_HALF_1,n.rectAreaLTC2=we.LTC_HALF_2)),n.ambient[0]=f,n.ambient[1]=h,n.ambient[2]=u;const v=n.hash;(v.directionalLength!==d||v.pointLength!==x||v.spotLength!==E||v.rectAreaLength!==m||v.hemiLength!==p||v.numDirectionalShadows!==T||v.numPointShadows!==P||v.numSpotShadows!==S||v.numSpotMaps!==C||v.numLightProbes!==I)&&(n.directional.length=d,n.spot.length=E,n.rectArea.length=m,n.point.length=x,n.hemi.length=p,n.directionalShadow.length=T,n.directionalShadowMap.length=T,n.pointShadow.length=P,n.pointShadowMap.length=P,n.spotShadow.length=S,n.spotShadowMap.length=S,n.directionalShadowMatrix.length=T,n.pointShadowMatrix.length=P,n.spotLightMatrix.length=S+C-y,n.spotLightMap.length=C,n.numSpotLightShadowsWithMaps=y,n.numLightProbes=I,v.directionalLength=d,v.pointLength=x,v.spotLength=E,v.rectAreaLength=m,v.hemiLength=p,v.numDirectionalShadows=T,v.numPointShadows=P,v.numSpotShadows=S,v.numSpotMaps=C,v.numLightProbes=I,n.version=WM++)}function l(c,f){let h=0,u=0,d=0,x=0,E=0;const m=f.matrixWorldInverse;for(let p=0,T=c.length;p<T;p++){const P=c[p];if(P.isDirectionalLight){const S=n.directional[h];S.direction.setFromMatrixPosition(P.matrixWorld),r.setFromMatrixPosition(P.target.matrixWorld),S.direction.sub(r),S.direction.transformDirection(m),h++}else if(P.isSpotLight){const S=n.spot[d];S.position.setFromMatrixPosition(P.matrixWorld),S.position.applyMatrix4(m),S.direction.setFromMatrixPosition(P.matrixWorld),r.setFromMatrixPosition(P.target.matrixWorld),S.direction.sub(r),S.direction.transformDirection(m),d++}else if(P.isRectAreaLight){const S=n.rectArea[x];S.position.setFromMatrixPosition(P.matrixWorld),S.position.applyMatrix4(m),a.identity(),s.copy(P.matrixWorld),s.premultiply(m),a.extractRotation(s),S.halfWidth.set(P.width*.5,0,0),S.halfHeight.set(0,P.height*.5,0),S.halfWidth.applyMatrix4(a),S.halfHeight.applyMatrix4(a),x++}else if(P.isPointLight){const S=n.point[u];S.position.setFromMatrixPosition(P.matrixWorld),S.position.applyMatrix4(m),u++}else if(P.isHemisphereLight){const S=n.hemi[E];S.direction.setFromMatrixPosition(P.matrixWorld),S.direction.transformDirection(m),E++}}}return{setup:o,setupView:l,state:n}}function Oh(i){const e=new YM(i),t=[],n=[],r=[];function s(u){h.camera=u,t.length=0,n.length=0,r.length=0}function a(u){t.push(u)}function o(u){n.push(u)}function l(u){r.push(u)}function c(){e.setup(t)}function f(u){e.setupView(t,u)}const h={lightsArray:t,shadowsArray:n,lightProbeGridArray:r,camera:null,lights:e,transmissionRenderTarget:{},textureUnits:0};return{init:s,state:h,setupLights:c,setupLightsView:f,pushLight:a,pushShadow:o,pushLightProbeGrid:l}}function qM(i){let e=new WeakMap;function t(r,s=0){const a=e.get(r);let o;return a===void 0?(o=new Oh(i),e.set(r,[o])):s>=a.length?(o=new Oh(i),a.push(o)):o=a[s],o}function n(){e=new WeakMap}return{get:t,dispose:n}}const KM=`void main() {
	gl_Position = vec4( position, 1.0 );
}`,$M=`uniform sampler2D shadow_pass;
uniform vec2 resolution;
uniform float radius;
void main() {
	const float samples = float( VSM_SAMPLES );
	float mean = 0.0;
	float squared_mean = 0.0;
	float uvStride = samples <= 1.0 ? 0.0 : 2.0 / ( samples - 1.0 );
	float uvStart = samples <= 1.0 ? 0.0 : - 1.0;
	for ( float i = 0.0; i < samples; i ++ ) {
		float uvOffset = uvStart + i * uvStride;
		#ifdef HORIZONTAL_PASS
			vec2 distribution = texture2D( shadow_pass, ( gl_FragCoord.xy + vec2( uvOffset, 0.0 ) * radius ) / resolution ).rg;
			mean += distribution.x;
			squared_mean += distribution.y * distribution.y + distribution.x * distribution.x;
		#else
			float depth = texture2D( shadow_pass, ( gl_FragCoord.xy + vec2( 0.0, uvOffset ) * radius ) / resolution ).r;
			mean += depth;
			squared_mean += depth * depth;
		#endif
	}
	mean = mean / samples;
	squared_mean = squared_mean / samples;
	float std_dev = sqrt( max( 0.0, squared_mean - mean * mean ) );
	gl_FragColor = vec4( mean, std_dev, 0.0, 1.0 );
}`,ZM=[new X(1,0,0),new X(-1,0,0),new X(0,1,0),new X(0,-1,0),new X(0,0,1),new X(0,0,-1)],JM=[new X(0,-1,0),new X(0,-1,0),new X(0,0,1),new X(0,0,-1),new X(0,-1,0),new X(0,-1,0)],Bh=new Xt,oa=new X,zl=new X;function jM(i,e,t){let n=new gu;const r=new $e,s=new $e,a=new $t,o=new a_,l=new o_,c={},f=t.maxTextureSize,h={[xr]:Vn,[Vn]:xr,[Pi]:Pi},u=new Oi({defines:{VSM_SAMPLES:8},uniforms:{shadow_pass:{value:null},resolution:{value:new $e},radius:{value:4}},vertexShader:KM,fragmentShader:$M}),d=u.clone();d.defines.HORIZONTAL_PASS=1;const x=new gn;x.setAttribute("position",new Qn(new Float32Array([-1,-1,.5,3,-1,.5,-1,3,.5]),3));const E=new vi(x,u),m=this;this.enabled=!1,this.autoUpdate=!0,this.needsUpdate=!1,this.type=oo;let p=this.type;this.render=function(y,I,v){if(m.enabled===!1||m.autoUpdate===!1&&m.needsUpdate===!1||y.length===0)return;this.type===Yg&&(Je("WebGLShadowMap: PCFSoftShadowMap has been deprecated. Using PCFShadowMap instead."),this.type=oo);const A=i.getRenderTarget(),F=i.getActiveCubeFace(),D=i.getActiveMipmapLevel(),z=i.state;z.setBlending(Ki),z.buffers.depth.getReversed()===!0?z.buffers.color.setClear(0,0,0,0):z.buffers.color.setClear(1,1,1,1),z.buffers.depth.setTest(!0),z.setScissorTest(!1);const O=p!==this.type;O&&I.traverse(function(k){k.material&&(Array.isArray(k.material)?k.material.forEach(L=>L.needsUpdate=!0):k.material.needsUpdate=!0)});for(let k=0,L=y.length;k<L;k++){const N=y[k],B=N.shadow;if(B===void 0){Je("WebGLShadowMap:",N,"has no shadow.");continue}if(B.autoUpdate===!1&&B.needsUpdate===!1)continue;r.copy(B.mapSize);const j=B.getFrameExtents();r.multiply(j),s.copy(B.mapSize),(r.x>f||r.y>f)&&(r.x>f&&(s.x=Math.floor(f/j.x),r.x=s.x*j.x,B.mapSize.x=s.x),r.y>f&&(s.y=Math.floor(f/j.y),r.y=s.y*j.y,B.mapSize.y=s.y));const ne=i.state.buffers.depth.getReversed();if(B.camera._reversedDepth=ne,B.map===null||O===!0){if(B.map!==null&&(B.map.depthTexture!==null&&(B.map.depthTexture.dispose(),B.map.depthTexture=null),B.map.dispose()),this.type===fa){if(N.isPointLight){Je("WebGLShadowMap: VSM shadow maps are not supported for PointLights. Use PCF or BasicShadowMap instead.");continue}B.map=new Ni(r.x,r.y,{format:Vr,type:Zi,minFilter:yn,magFilter:yn,generateMipmaps:!1}),B.map.texture.name=N.name+".shadowMap",B.map.depthTexture=new Is(r.x,r.y,Li),B.map.depthTexture.name=N.name+".shadowMapDepth",B.map.depthTexture.format=Ji,B.map.depthTexture.compareFunction=null,B.map.depthTexture.minFilter=Mn,B.map.depthTexture.magFilter=Mn}else N.isPointLight?(B.map=new lp(r.x),B.map.depthTexture=new e_(r.x,Fi)):(B.map=new Ni(r.x,r.y),B.map.depthTexture=new Is(r.x,r.y,Fi)),B.map.depthTexture.name=N.name+".shadowMap",B.map.depthTexture.format=Ji,this.type===oo?(B.map.depthTexture.compareFunction=ne?hu:fu,B.map.depthTexture.minFilter=yn,B.map.depthTexture.magFilter=yn):(B.map.depthTexture.compareFunction=null,B.map.depthTexture.minFilter=Mn,B.map.depthTexture.magFilter=Mn);B.camera.updateProjectionMatrix()}const ae=B.map.isWebGLCubeRenderTarget?6:1;for(let fe=0;fe<ae;fe++){if(B.map.isWebGLCubeRenderTarget)i.setRenderTarget(B.map,fe),i.clear();else{fe===0&&(i.setRenderTarget(B.map),i.clear());const re=B.getViewport(fe);a.set(s.x*re.x,s.y*re.y,s.x*re.z,s.y*re.w),z.viewport(a)}if(N.isPointLight){const re=B.camera,le=B.matrix,Ze=N.distance||re.far;Ze!==re.far&&(re.far=Ze,re.updateProjectionMatrix()),oa.setFromMatrixPosition(N.matrixWorld),re.position.copy(oa),zl.copy(re.position),zl.add(ZM[fe]),re.up.copy(JM[fe]),re.lookAt(zl),re.updateMatrixWorld(),le.makeTranslation(-oa.x,-oa.y,-oa.z),Bh.multiplyMatrices(re.projectionMatrix,re.matrixWorldInverse),B._frustum.setFromProjectionMatrix(Bh,re.coordinateSystem,re.reversedDepth)}else B.updateMatrices(N);n=B.getFrustum(),S(I,v,B.camera,N,this.type)}B.isPointLightShadow!==!0&&this.type===fa&&T(B,v),B.needsUpdate=!1}p=this.type,m.needsUpdate=!1,i.setRenderTarget(A,F,D)};function T(y,I){const v=e.update(E);u.defines.VSM_SAMPLES!==y.blurSamples&&(u.defines.VSM_SAMPLES=y.blurSamples,d.defines.VSM_SAMPLES=y.blurSamples,u.needsUpdate=!0,d.needsUpdate=!0),y.mapPass===null&&(y.mapPass=new Ni(r.x,r.y,{format:Vr,type:Zi})),u.uniforms.shadow_pass.value=y.map.depthTexture,u.uniforms.resolution.value=y.mapSize,u.uniforms.radius.value=y.radius,i.setRenderTarget(y.mapPass),i.clear(),i.renderBufferDirect(I,null,v,u,E,null),d.uniforms.shadow_pass.value=y.mapPass.texture,d.uniforms.resolution.value=y.mapSize,d.uniforms.radius.value=y.radius,i.setRenderTarget(y.map),i.clear(),i.renderBufferDirect(I,null,v,d,E,null)}function P(y,I,v,A){let F=null;const D=v.isPointLight===!0?y.customDistanceMaterial:y.customDepthMaterial;if(D!==void 0)F=D;else if(F=v.isPointLight===!0?l:o,i.localClippingEnabled&&I.clipShadows===!0&&Array.isArray(I.clippingPlanes)&&I.clippingPlanes.length!==0||I.displacementMap&&I.displacementScale!==0||I.alphaMap&&I.alphaTest>0||I.map&&I.alphaTest>0||I.alphaToCoverage===!0){const z=F.uuid,O=I.uuid;let k=c[z];k===void 0&&(k={},c[z]=k);let L=k[O];L===void 0&&(L=F.clone(),k[O]=L,I.addEventListener("dispose",C)),F=L}if(F.visible=I.visible,F.wireframe=I.wireframe,A===fa?F.side=I.shadowSide!==null?I.shadowSide:I.side:F.side=I.shadowSide!==null?I.shadowSide:h[I.side],F.alphaMap=I.alphaMap,F.alphaTest=I.alphaToCoverage===!0?.5:I.alphaTest,F.map=I.map,F.clipShadows=I.clipShadows,F.clippingPlanes=I.clippingPlanes,F.clipIntersection=I.clipIntersection,F.displacementMap=I.displacementMap,F.displacementScale=I.displacementScale,F.displacementBias=I.displacementBias,F.wireframeLinewidth=I.wireframeLinewidth,F.linewidth=I.linewidth,v.isPointLight===!0&&F.isMeshDistanceMaterial===!0){const z=i.properties.get(F);z.light=v}return F}function S(y,I,v,A,F){if(y.visible===!1)return;if(y.layers.test(I.layers)&&(y.isMesh||y.isLine||y.isPoints)&&(y.castShadow||y.receiveShadow&&F===fa)&&(!y.frustumCulled||n.intersectsObject(y))){y.modelViewMatrix.multiplyMatrices(v.matrixWorldInverse,y.matrixWorld);const O=e.update(y),k=y.material;if(Array.isArray(k)){const L=O.groups;for(let N=0,B=L.length;N<B;N++){const j=L[N],ne=k[j.materialIndex];if(ne&&ne.visible){const ae=P(y,ne,A,F);y.onBeforeShadow(i,y,I,v,O,ae,j),i.renderBufferDirect(v,null,O,ae,y,j),y.onAfterShadow(i,y,I,v,O,ae,j)}}}else if(k.visible){const L=P(y,k,A,F);y.onBeforeShadow(i,y,I,v,O,L,null),i.renderBufferDirect(v,null,O,L,y,null),y.onAfterShadow(i,y,I,v,O,L,null)}}const z=y.children;for(let O=0,k=z.length;O<k;O++)S(z[O],I,v,A,F)}function C(y){y.target.removeEventListener("dispose",C);for(const v in c){const A=c[v],F=y.target.uuid;F in A&&(A[F].dispose(),delete A[F])}}}function QM(i,e){function t(){let G=!1;const Me=new $t;let oe=null;const Ee=new $t(0,0,0,0);return{setMask:function(Re){oe!==Re&&!G&&(i.colorMask(Re,Re,Re,Re),oe=Re)},setLocked:function(Re){G=Re},setClear:function(Re,de,Ge,Ne,at){at===!0&&(Re*=Ne,de*=Ne,Ge*=Ne),Me.set(Re,de,Ge,Ne),Ee.equals(Me)===!1&&(i.clearColor(Re,de,Ge,Ne),Ee.copy(Me))},reset:function(){G=!1,oe=null,Ee.set(-1,0,0,0)}}}function n(){let G=!1,Me=!1,oe=null,Ee=null,Re=null;return{setReversed:function(de){if(Me!==de){const Ge=e.get("EXT_clip_control");de?Ge.clipControlEXT(Ge.LOWER_LEFT_EXT,Ge.ZERO_TO_ONE_EXT):Ge.clipControlEXT(Ge.LOWER_LEFT_EXT,Ge.NEGATIVE_ONE_TO_ONE_EXT),Me=de;const Ne=Re;Re=null,this.setClear(Ne)}},getReversed:function(){return Me},setTest:function(de){de?he(i.DEPTH_TEST):Be(i.DEPTH_TEST)},setMask:function(de){oe!==de&&!G&&(i.depthMask(de),oe=de)},setFunc:function(de){if(Me&&(de=T0[de]),Ee!==de){switch(de){case tc:i.depthFunc(i.NEVER);break;case nc:i.depthFunc(i.ALWAYS);break;case ic:i.depthFunc(i.LESS);break;case Ds:i.depthFunc(i.LEQUAL);break;case rc:i.depthFunc(i.EQUAL);break;case sc:i.depthFunc(i.GEQUAL);break;case ac:i.depthFunc(i.GREATER);break;case oc:i.depthFunc(i.NOTEQUAL);break;default:i.depthFunc(i.LEQUAL)}Ee=de}},setLocked:function(de){G=de},setClear:function(de){Re!==de&&(Re=de,Me&&(de=1-de),i.clearDepth(de))},reset:function(){G=!1,oe=null,Ee=null,Re=null,Me=!1}}}function r(){let G=!1,Me=null,oe=null,Ee=null,Re=null,de=null,Ge=null,Ne=null,at=null;return{setTest:function(lt){G||(lt?he(i.STENCIL_TEST):Be(i.STENCIL_TEST))},setMask:function(lt){Me!==lt&&!G&&(i.stencilMask(lt),Me=lt)},setFunc:function(lt,xn,vn){(oe!==lt||Ee!==xn||Re!==vn)&&(i.stencilFunc(lt,xn,vn),oe=lt,Ee=xn,Re=vn)},setOp:function(lt,xn,vn){(de!==lt||Ge!==xn||Ne!==vn)&&(i.stencilOp(lt,xn,vn),de=lt,Ge=xn,Ne=vn)},setLocked:function(lt){G=lt},setClear:function(lt){at!==lt&&(i.clearStencil(lt),at=lt)},reset:function(){G=!1,Me=null,oe=null,Ee=null,Re=null,de=null,Ge=null,Ne=null,at=null}}}const s=new t,a=new n,o=new r,l=new WeakMap,c=new WeakMap;let f={},h={},u={},d=new WeakMap,x=[],E=null,m=!1,p=null,T=null,P=null,S=null,C=null,y=null,I=null,v=new ft(0,0,0),A=0,F=!1,D=null,z=null,O=null,k=null,L=null;const N=i.getParameter(i.MAX_COMBINED_TEXTURE_IMAGE_UNITS);let B=!1,j=0;const ne=i.getParameter(i.VERSION);ne.indexOf("WebGL")!==-1?(j=parseFloat(/^WebGL (\d)/.exec(ne)[1]),B=j>=1):ne.indexOf("OpenGL ES")!==-1&&(j=parseFloat(/^OpenGL ES (\d)/.exec(ne)[1]),B=j>=2);let ae=null,fe={};const re=i.getParameter(i.SCISSOR_BOX),le=i.getParameter(i.VIEWPORT),Ze=new $t().fromArray(re),Ke=new $t().fromArray(le);function Q(G,Me,oe,Ee){const Re=new Uint8Array(4),de=i.createTexture();i.bindTexture(G,de),i.texParameteri(G,i.TEXTURE_MIN_FILTER,i.NEAREST),i.texParameteri(G,i.TEXTURE_MAG_FILTER,i.NEAREST);for(let Ge=0;Ge<oe;Ge++)G===i.TEXTURE_3D||G===i.TEXTURE_2D_ARRAY?i.texImage3D(Me,0,i.RGBA,1,1,Ee,0,i.RGBA,i.UNSIGNED_BYTE,Re):i.texImage2D(Me+Ge,0,i.RGBA,1,1,0,i.RGBA,i.UNSIGNED_BYTE,Re);return de}const ue={};ue[i.TEXTURE_2D]=Q(i.TEXTURE_2D,i.TEXTURE_2D,1),ue[i.TEXTURE_CUBE_MAP]=Q(i.TEXTURE_CUBE_MAP,i.TEXTURE_CUBE_MAP_POSITIVE_X,6),ue[i.TEXTURE_2D_ARRAY]=Q(i.TEXTURE_2D_ARRAY,i.TEXTURE_2D_ARRAY,1,1),ue[i.TEXTURE_3D]=Q(i.TEXTURE_3D,i.TEXTURE_3D,1,1),s.setClear(0,0,0,1),a.setClear(1),o.setClear(0),he(i.DEPTH_TEST),a.setFunc(Ds),Ft(!1),Gt(If),he(i.CULL_FACE),ot(Ki);function he(G){f[G]!==!0&&(i.enable(G),f[G]=!0)}function Be(G){f[G]!==!1&&(i.disable(G),f[G]=!1)}function Ue(G,Me){return u[G]!==Me?(i.bindFramebuffer(G,Me),u[G]=Me,G===i.DRAW_FRAMEBUFFER&&(u[i.FRAMEBUFFER]=Me),G===i.FRAMEBUFFER&&(u[i.DRAW_FRAMEBUFFER]=Me),!0):!1}function ze(G,Me){let oe=x,Ee=!1;if(G){oe=d.get(Me),oe===void 0&&(oe=[],d.set(Me,oe));const Re=G.textures;if(oe.length!==Re.length||oe[0]!==i.COLOR_ATTACHMENT0){for(let de=0,Ge=Re.length;de<Ge;de++)oe[de]=i.COLOR_ATTACHMENT0+de;oe.length=Re.length,Ee=!0}}else oe[0]!==i.BACK&&(oe[0]=i.BACK,Ee=!0);Ee&&i.drawBuffers(oe)}function St(G){return E!==G?(i.useProgram(G),E=G,!0):!1}const je={[Ir]:i.FUNC_ADD,[Kg]:i.FUNC_SUBTRACT,[$g]:i.FUNC_REVERSE_SUBTRACT};je[Zg]=i.MIN,je[Jg]=i.MAX;const ht={[jg]:i.ZERO,[Qg]:i.ONE,[e0]:i.SRC_COLOR,[Ql]:i.SRC_ALPHA,[a0]:i.SRC_ALPHA_SATURATE,[r0]:i.DST_COLOR,[n0]:i.DST_ALPHA,[t0]:i.ONE_MINUS_SRC_COLOR,[ec]:i.ONE_MINUS_SRC_ALPHA,[s0]:i.ONE_MINUS_DST_COLOR,[i0]:i.ONE_MINUS_DST_ALPHA,[o0]:i.CONSTANT_COLOR,[l0]:i.ONE_MINUS_CONSTANT_COLOR,[c0]:i.CONSTANT_ALPHA,[u0]:i.ONE_MINUS_CONSTANT_ALPHA};function ot(G,Me,oe,Ee,Re,de,Ge,Ne,at,lt){if(G===Ki){m===!0&&(Be(i.BLEND),m=!1);return}if(m===!1&&(he(i.BLEND),m=!0),G!==qg){if(G!==p||lt!==F){if((T!==Ir||C!==Ir)&&(i.blendEquation(i.FUNC_ADD),T=Ir,C=Ir),lt)switch(G){case bs:i.blendFuncSeparate(i.ONE,i.ONE_MINUS_SRC_ALPHA,i.ONE,i.ONE_MINUS_SRC_ALPHA);break;case Uf:i.blendFunc(i.ONE,i.ONE);break;case Nf:i.blendFuncSeparate(i.ZERO,i.ONE_MINUS_SRC_COLOR,i.ZERO,i.ONE);break;case Ff:i.blendFuncSeparate(i.DST_COLOR,i.ONE_MINUS_SRC_ALPHA,i.ZERO,i.ONE);break;default:gt("WebGLState: Invalid blending: ",G);break}else switch(G){case bs:i.blendFuncSeparate(i.SRC_ALPHA,i.ONE_MINUS_SRC_ALPHA,i.ONE,i.ONE_MINUS_SRC_ALPHA);break;case Uf:i.blendFuncSeparate(i.SRC_ALPHA,i.ONE,i.ONE,i.ONE);break;case Nf:gt("WebGLState: SubtractiveBlending requires material.premultipliedAlpha = true");break;case Ff:gt("WebGLState: MultiplyBlending requires material.premultipliedAlpha = true");break;default:gt("WebGLState: Invalid blending: ",G);break}P=null,S=null,y=null,I=null,v.set(0,0,0),A=0,p=G,F=lt}return}Re=Re||Me,de=de||oe,Ge=Ge||Ee,(Me!==T||Re!==C)&&(i.blendEquationSeparate(je[Me],je[Re]),T=Me,C=Re),(oe!==P||Ee!==S||de!==y||Ge!==I)&&(i.blendFuncSeparate(ht[oe],ht[Ee],ht[de],ht[Ge]),P=oe,S=Ee,y=de,I=Ge),(Ne.equals(v)===!1||at!==A)&&(i.blendColor(Ne.r,Ne.g,Ne.b,at),v.copy(Ne),A=at),p=G,F=!1}function rt(G,Me){G.side===Pi?Be(i.CULL_FACE):he(i.CULL_FACE);let oe=G.side===Vn;Me&&(oe=!oe),Ft(oe),G.blending===bs&&G.transparent===!1?ot(Ki):ot(G.blending,G.blendEquation,G.blendSrc,G.blendDst,G.blendEquationAlpha,G.blendSrcAlpha,G.blendDstAlpha,G.blendColor,G.blendAlpha,G.premultipliedAlpha),a.setFunc(G.depthFunc),a.setTest(G.depthTest),a.setMask(G.depthWrite),s.setMask(G.colorWrite);const Ee=G.stencilWrite;o.setTest(Ee),Ee&&(o.setMask(G.stencilWriteMask),o.setFunc(G.stencilFunc,G.stencilRef,G.stencilFuncMask),o.setOp(G.stencilFail,G.stencilZFail,G.stencilZPass)),_t(G.polygonOffset,G.polygonOffsetFactor,G.polygonOffsetUnits),G.alphaToCoverage===!0?he(i.SAMPLE_ALPHA_TO_COVERAGE):Be(i.SAMPLE_ALPHA_TO_COVERAGE)}function Ft(G){D!==G&&(G?i.frontFace(i.CW):i.frontFace(i.CCW),D=G)}function Gt(G){G!==Wg?(he(i.CULL_FACE),G!==z&&(G===If?i.cullFace(i.BACK):G===Xg?i.cullFace(i.FRONT):i.cullFace(i.FRONT_AND_BACK))):Be(i.CULL_FACE),z=G}function Ot(G){G!==O&&(B&&i.lineWidth(G),O=G)}function _t(G,Me,oe){G?(he(i.POLYGON_OFFSET_FILL),(k!==Me||L!==oe)&&(k=Me,L=oe,a.getReversed()&&(Me=-Me),i.polygonOffset(Me,oe))):Be(i.POLYGON_OFFSET_FILL)}function wt(G){G?he(i.SCISSOR_TEST):Be(i.SCISSOR_TEST)}function Tt(G){G===void 0&&(G=i.TEXTURE0+N-1),ae!==G&&(i.activeTexture(G),ae=G)}function H(G,Me,oe){oe===void 0&&(ae===null?oe=i.TEXTURE0+N-1:oe=ae);let Ee=fe[oe];Ee===void 0&&(Ee={type:void 0,texture:void 0},fe[oe]=Ee),(Ee.type!==G||Ee.texture!==Me)&&(ae!==oe&&(i.activeTexture(oe),ae=oe),i.bindTexture(G,Me||ue[G]),Ee.type=G,Ee.texture=Me)}function Xe(){const G=fe[ae];G!==void 0&&G.type!==void 0&&(i.bindTexture(G.type,null),G.type=void 0,G.texture=void 0)}function pe(){try{i.compressedTexImage2D(...arguments)}catch(G){gt("WebGLState:",G)}}function w(){try{i.compressedTexImage3D(...arguments)}catch(G){gt("WebGLState:",G)}}function _(){try{i.texSubImage2D(...arguments)}catch(G){gt("WebGLState:",G)}}function Y(){try{i.texSubImage3D(...arguments)}catch(G){gt("WebGLState:",G)}}function $(){try{i.compressedTexSubImage2D(...arguments)}catch(G){gt("WebGLState:",G)}}function ee(){try{i.compressedTexSubImage3D(...arguments)}catch(G){gt("WebGLState:",G)}}function ge(){try{i.texStorage2D(...arguments)}catch(G){gt("WebGLState:",G)}}function _e(){try{i.texStorage3D(...arguments)}catch(G){gt("WebGLState:",G)}}function te(){try{i.texImage2D(...arguments)}catch(G){gt("WebGLState:",G)}}function ie(){try{i.texImage3D(...arguments)}catch(G){gt("WebGLState:",G)}}function xe(G){return h[G]!==void 0?h[G]:i.getParameter(G)}function Ve(G,Me){h[G]!==Me&&(i.pixelStorei(G,Me),h[G]=Me)}function ye(G){Ze.equals(G)===!1&&(i.scissor(G.x,G.y,G.z,G.w),Ze.copy(G))}function Se(G){Ke.equals(G)===!1&&(i.viewport(G.x,G.y,G.z,G.w),Ke.copy(G))}function ke(G,Me){let oe=c.get(Me);oe===void 0&&(oe=new WeakMap,c.set(Me,oe));let Ee=oe.get(G);Ee===void 0&&(Ee=i.getUniformBlockIndex(Me,G.name),oe.set(G,Ee))}function Ye(G,Me){const Ee=c.get(Me).get(G);l.get(Me)!==Ee&&(i.uniformBlockBinding(Me,Ee,G.__bindingPointIndex),l.set(Me,Ee))}function qe(){i.disable(i.BLEND),i.disable(i.CULL_FACE),i.disable(i.DEPTH_TEST),i.disable(i.POLYGON_OFFSET_FILL),i.disable(i.SCISSOR_TEST),i.disable(i.STENCIL_TEST),i.disable(i.SAMPLE_ALPHA_TO_COVERAGE),i.blendEquation(i.FUNC_ADD),i.blendFunc(i.ONE,i.ZERO),i.blendFuncSeparate(i.ONE,i.ZERO,i.ONE,i.ZERO),i.blendColor(0,0,0,0),i.colorMask(!0,!0,!0,!0),i.clearColor(0,0,0,0),i.depthMask(!0),i.depthFunc(i.LESS),a.setReversed(!1),i.clearDepth(1),i.stencilMask(4294967295),i.stencilFunc(i.ALWAYS,0,4294967295),i.stencilOp(i.KEEP,i.KEEP,i.KEEP),i.clearStencil(0),i.cullFace(i.BACK),i.frontFace(i.CCW),i.polygonOffset(0,0),i.activeTexture(i.TEXTURE0),i.bindFramebuffer(i.FRAMEBUFFER,null),i.bindFramebuffer(i.DRAW_FRAMEBUFFER,null),i.bindFramebuffer(i.READ_FRAMEBUFFER,null),i.useProgram(null),i.lineWidth(1),i.scissor(0,0,i.canvas.width,i.canvas.height),i.viewport(0,0,i.canvas.width,i.canvas.height),i.pixelStorei(i.PACK_ALIGNMENT,4),i.pixelStorei(i.UNPACK_ALIGNMENT,4),i.pixelStorei(i.UNPACK_FLIP_Y_WEBGL,!1),i.pixelStorei(i.UNPACK_PREMULTIPLY_ALPHA_WEBGL,!1),i.pixelStorei(i.UNPACK_COLORSPACE_CONVERSION_WEBGL,i.BROWSER_DEFAULT_WEBGL),i.pixelStorei(i.PACK_ROW_LENGTH,0),i.pixelStorei(i.PACK_SKIP_PIXELS,0),i.pixelStorei(i.PACK_SKIP_ROWS,0),i.pixelStorei(i.UNPACK_ROW_LENGTH,0),i.pixelStorei(i.UNPACK_IMAGE_HEIGHT,0),i.pixelStorei(i.UNPACK_SKIP_PIXELS,0),i.pixelStorei(i.UNPACK_SKIP_ROWS,0),i.pixelStorei(i.UNPACK_SKIP_IMAGES,0),f={},h={},ae=null,fe={},u={},d=new WeakMap,x=[],E=null,m=!1,p=null,T=null,P=null,S=null,C=null,y=null,I=null,v=new ft(0,0,0),A=0,F=!1,D=null,z=null,O=null,k=null,L=null,Ze.set(0,0,i.canvas.width,i.canvas.height),Ke.set(0,0,i.canvas.width,i.canvas.height),s.reset(),a.reset(),o.reset()}return{buffers:{color:s,depth:a,stencil:o},enable:he,disable:Be,bindFramebuffer:Ue,drawBuffers:ze,useProgram:St,setBlending:ot,setMaterial:rt,setFlipSided:Ft,setCullFace:Gt,setLineWidth:Ot,setPolygonOffset:_t,setScissorTest:wt,activeTexture:Tt,bindTexture:H,unbindTexture:Xe,compressedTexImage2D:pe,compressedTexImage3D:w,texImage2D:te,texImage3D:ie,pixelStorei:Ve,getParameter:xe,updateUBOMapping:ke,uniformBlockBinding:Ye,texStorage2D:ge,texStorage3D:_e,texSubImage2D:_,texSubImage3D:Y,compressedTexSubImage2D:$,compressedTexSubImage3D:ee,scissor:ye,viewport:Se,reset:qe}}function ey(i,e,t,n,r,s,a){const o=e.has("WEBGL_multisampled_render_to_texture")?e.get("WEBGL_multisampled_render_to_texture"):null,l=typeof navigator>"u"?!1:/OculusBrowser/g.test(navigator.userAgent),c=new $e,f=new WeakMap,h=new Set;let u;const d=new WeakMap;let x=!1;try{x=typeof OffscreenCanvas<"u"&&new OffscreenCanvas(1,1).getContext("2d")!==null}catch{}function E(w,_){return x?new OffscreenCanvas(w,_):Eo("canvas")}function m(w,_,Y){let $=1;const ee=pe(w);if((ee.width>Y||ee.height>Y)&&($=Y/Math.max(ee.width,ee.height)),$<1)if(typeof HTMLImageElement<"u"&&w instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&w instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&w instanceof ImageBitmap||typeof VideoFrame<"u"&&w instanceof VideoFrame){const ge=Math.floor($*ee.width),_e=Math.floor($*ee.height);u===void 0&&(u=E(ge,_e));const te=_?E(ge,_e):u;return te.width=ge,te.height=_e,te.getContext("2d").drawImage(w,0,0,ge,_e),Je("WebGLRenderer: Texture has been resized from ("+ee.width+"x"+ee.height+") to ("+ge+"x"+_e+")."),te}else return"data"in w&&Je("WebGLRenderer: Image in DataTexture is too big ("+ee.width+"x"+ee.height+")."),w;return w}function p(w){return w.generateMipmaps}function T(w){i.generateMipmap(w)}function P(w){return w.isWebGLCubeRenderTarget?i.TEXTURE_CUBE_MAP:w.isWebGL3DRenderTarget?i.TEXTURE_3D:w.isWebGLArrayRenderTarget||w.isCompressedArrayTexture?i.TEXTURE_2D_ARRAY:i.TEXTURE_2D}function S(w,_,Y,$,ee,ge=!1){if(w!==null){if(i[w]!==void 0)return i[w];Je("WebGLRenderer: Attempt to use non-existing WebGL internal format '"+w+"'")}let _e;$&&(_e=e.get("EXT_texture_norm16"),_e||Je("WebGLRenderer: Unable to use normalized textures without EXT_texture_norm16 extension"));let te=_;if(_===i.RED&&(Y===i.FLOAT&&(te=i.R32F),Y===i.HALF_FLOAT&&(te=i.R16F),Y===i.UNSIGNED_BYTE&&(te=i.R8),Y===i.UNSIGNED_SHORT&&_e&&(te=_e.R16_EXT),Y===i.SHORT&&_e&&(te=_e.R16_SNORM_EXT)),_===i.RED_INTEGER&&(Y===i.UNSIGNED_BYTE&&(te=i.R8UI),Y===i.UNSIGNED_SHORT&&(te=i.R16UI),Y===i.UNSIGNED_INT&&(te=i.R32UI),Y===i.BYTE&&(te=i.R8I),Y===i.SHORT&&(te=i.R16I),Y===i.INT&&(te=i.R32I)),_===i.RG&&(Y===i.FLOAT&&(te=i.RG32F),Y===i.HALF_FLOAT&&(te=i.RG16F),Y===i.UNSIGNED_BYTE&&(te=i.RG8),Y===i.UNSIGNED_SHORT&&_e&&(te=_e.RG16_EXT),Y===i.SHORT&&_e&&(te=_e.RG16_SNORM_EXT)),_===i.RG_INTEGER&&(Y===i.UNSIGNED_BYTE&&(te=i.RG8UI),Y===i.UNSIGNED_SHORT&&(te=i.RG16UI),Y===i.UNSIGNED_INT&&(te=i.RG32UI),Y===i.BYTE&&(te=i.RG8I),Y===i.SHORT&&(te=i.RG16I),Y===i.INT&&(te=i.RG32I)),_===i.RGB_INTEGER&&(Y===i.UNSIGNED_BYTE&&(te=i.RGB8UI),Y===i.UNSIGNED_SHORT&&(te=i.RGB16UI),Y===i.UNSIGNED_INT&&(te=i.RGB32UI),Y===i.BYTE&&(te=i.RGB8I),Y===i.SHORT&&(te=i.RGB16I),Y===i.INT&&(te=i.RGB32I)),_===i.RGBA_INTEGER&&(Y===i.UNSIGNED_BYTE&&(te=i.RGBA8UI),Y===i.UNSIGNED_SHORT&&(te=i.RGBA16UI),Y===i.UNSIGNED_INT&&(te=i.RGBA32UI),Y===i.BYTE&&(te=i.RGBA8I),Y===i.SHORT&&(te=i.RGBA16I),Y===i.INT&&(te=i.RGBA32I)),_===i.RGB&&(Y===i.UNSIGNED_SHORT&&_e&&(te=_e.RGB16_EXT),Y===i.SHORT&&_e&&(te=_e.RGB16_SNORM_EXT),Y===i.UNSIGNED_INT_5_9_9_9_REV&&(te=i.RGB9_E5),Y===i.UNSIGNED_INT_10F_11F_11F_REV&&(te=i.R11F_G11F_B10F)),_===i.RGBA){const ie=ge?yo:mt.getTransfer(ee);Y===i.FLOAT&&(te=i.RGBA32F),Y===i.HALF_FLOAT&&(te=i.RGBA16F),Y===i.UNSIGNED_BYTE&&(te=ie===Ct?i.SRGB8_ALPHA8:i.RGBA8),Y===i.UNSIGNED_SHORT&&_e&&(te=_e.RGBA16_EXT),Y===i.SHORT&&_e&&(te=_e.RGBA16_SNORM_EXT),Y===i.UNSIGNED_SHORT_4_4_4_4&&(te=i.RGBA4),Y===i.UNSIGNED_SHORT_5_5_5_1&&(te=i.RGB5_A1)}return(te===i.R16F||te===i.R32F||te===i.RG16F||te===i.RG32F||te===i.RGBA16F||te===i.RGBA32F)&&e.get("EXT_color_buffer_float"),te}function C(w,_){let Y;return w?_===null||_===Fi||_===xa?Y=i.DEPTH24_STENCIL8:_===Li?Y=i.DEPTH32F_STENCIL8:_===_a&&(Y=i.DEPTH24_STENCIL8,Je("DepthTexture: 16 bit depth attachment is not supported with stencil. Using 24-bit attachment.")):_===null||_===Fi||_===xa?Y=i.DEPTH_COMPONENT24:_===Li?Y=i.DEPTH_COMPONENT32F:_===_a&&(Y=i.DEPTH_COMPONENT16),Y}function y(w,_){return p(w)===!0||w.isFramebufferTexture&&w.minFilter!==Mn&&w.minFilter!==yn?Math.log2(Math.max(_.width,_.height))+1:w.mipmaps!==void 0&&w.mipmaps.length>0?w.mipmaps.length:w.isCompressedTexture&&Array.isArray(w.image)?_.mipmaps.length:1}function I(w){const _=w.target;_.removeEventListener("dispose",I),A(_),_.isVideoTexture&&f.delete(_),_.isHTMLTexture&&h.delete(_)}function v(w){const _=w.target;_.removeEventListener("dispose",v),D(_)}function A(w){const _=n.get(w);if(_.__webglInit===void 0)return;const Y=w.source,$=d.get(Y);if($){const ee=$[_.__cacheKey];ee.usedTimes--,ee.usedTimes===0&&F(w),Object.keys($).length===0&&d.delete(Y)}n.remove(w)}function F(w){const _=n.get(w);i.deleteTexture(_.__webglTexture);const Y=w.source,$=d.get(Y);delete $[_.__cacheKey],a.memory.textures--}function D(w){const _=n.get(w);if(w.depthTexture&&(w.depthTexture.dispose(),n.remove(w.depthTexture)),w.isWebGLCubeRenderTarget)for(let $=0;$<6;$++){if(Array.isArray(_.__webglFramebuffer[$]))for(let ee=0;ee<_.__webglFramebuffer[$].length;ee++)i.deleteFramebuffer(_.__webglFramebuffer[$][ee]);else i.deleteFramebuffer(_.__webglFramebuffer[$]);_.__webglDepthbuffer&&i.deleteRenderbuffer(_.__webglDepthbuffer[$])}else{if(Array.isArray(_.__webglFramebuffer))for(let $=0;$<_.__webglFramebuffer.length;$++)i.deleteFramebuffer(_.__webglFramebuffer[$]);else i.deleteFramebuffer(_.__webglFramebuffer);if(_.__webglDepthbuffer&&i.deleteRenderbuffer(_.__webglDepthbuffer),_.__webglMultisampledFramebuffer&&i.deleteFramebuffer(_.__webglMultisampledFramebuffer),_.__webglColorRenderbuffer)for(let $=0;$<_.__webglColorRenderbuffer.length;$++)_.__webglColorRenderbuffer[$]&&i.deleteRenderbuffer(_.__webglColorRenderbuffer[$]);_.__webglDepthRenderbuffer&&i.deleteRenderbuffer(_.__webglDepthRenderbuffer)}const Y=w.textures;for(let $=0,ee=Y.length;$<ee;$++){const ge=n.get(Y[$]);ge.__webglTexture&&(i.deleteTexture(ge.__webglTexture),a.memory.textures--),n.remove(Y[$])}n.remove(w)}let z=0;function O(){z=0}function k(){return z}function L(w){z=w}function N(){const w=z;return w>=r.maxTextures&&Je("WebGLTextures: Trying to use "+w+" texture units while this GPU supports only "+r.maxTextures),z+=1,w}function B(w){const _=[];return _.push(w.wrapS),_.push(w.wrapT),_.push(w.wrapR||0),_.push(w.magFilter),_.push(w.minFilter),_.push(w.anisotropy),_.push(w.internalFormat),_.push(w.format),_.push(w.type),_.push(w.generateMipmaps),_.push(w.premultiplyAlpha),_.push(w.flipY),_.push(w.unpackAlignment),_.push(w.colorSpace),_.join()}function j(w,_){const Y=n.get(w);if(w.isVideoTexture&&H(w),w.isRenderTargetTexture===!1&&w.isExternalTexture!==!0&&w.version>0&&Y.__version!==w.version){const $=w.image;if($===null)Je("WebGLRenderer: Texture marked for update but no image data found.");else if($.complete===!1)Je("WebGLRenderer: Texture marked for update but image is incomplete");else{Be(Y,w,_);return}}else w.isExternalTexture&&(Y.__webglTexture=w.sourceTexture?w.sourceTexture:null);t.bindTexture(i.TEXTURE_2D,Y.__webglTexture,i.TEXTURE0+_)}function ne(w,_){const Y=n.get(w);if(w.isRenderTargetTexture===!1&&w.version>0&&Y.__version!==w.version){Be(Y,w,_);return}else w.isExternalTexture&&(Y.__webglTexture=w.sourceTexture?w.sourceTexture:null);t.bindTexture(i.TEXTURE_2D_ARRAY,Y.__webglTexture,i.TEXTURE0+_)}function ae(w,_){const Y=n.get(w);if(w.isRenderTargetTexture===!1&&w.version>0&&Y.__version!==w.version){Be(Y,w,_);return}t.bindTexture(i.TEXTURE_3D,Y.__webglTexture,i.TEXTURE0+_)}function fe(w,_){const Y=n.get(w);if(w.isCubeDepthTexture!==!0&&w.version>0&&Y.__version!==w.version){Ue(Y,w,_);return}t.bindTexture(i.TEXTURE_CUBE_MAP,Y.__webglTexture,i.TEXTURE0+_)}const re={[lc]:i.REPEAT,[Xi]:i.CLAMP_TO_EDGE,[cc]:i.MIRRORED_REPEAT},le={[Mn]:i.NEAREST,[d0]:i.NEAREST_MIPMAP_NEAREST,[Da]:i.NEAREST_MIPMAP_LINEAR,[yn]:i.LINEAR,[ol]:i.LINEAR_MIPMAP_NEAREST,[Fr]:i.LINEAR_MIPMAP_LINEAR},Ze={[g0]:i.NEVER,[M0]:i.ALWAYS,[_0]:i.LESS,[fu]:i.LEQUAL,[x0]:i.EQUAL,[hu]:i.GEQUAL,[v0]:i.GREATER,[S0]:i.NOTEQUAL};function Ke(w,_){if(_.type===Li&&e.has("OES_texture_float_linear")===!1&&(_.magFilter===yn||_.magFilter===ol||_.magFilter===Da||_.magFilter===Fr||_.minFilter===yn||_.minFilter===ol||_.minFilter===Da||_.minFilter===Fr)&&Je("WebGLRenderer: Unable to use linear filtering with floating point textures. OES_texture_float_linear not supported on this device."),i.texParameteri(w,i.TEXTURE_WRAP_S,re[_.wrapS]),i.texParameteri(w,i.TEXTURE_WRAP_T,re[_.wrapT]),(w===i.TEXTURE_3D||w===i.TEXTURE_2D_ARRAY)&&i.texParameteri(w,i.TEXTURE_WRAP_R,re[_.wrapR]),i.texParameteri(w,i.TEXTURE_MAG_FILTER,le[_.magFilter]),i.texParameteri(w,i.TEXTURE_MIN_FILTER,le[_.minFilter]),_.compareFunction&&(i.texParameteri(w,i.TEXTURE_COMPARE_MODE,i.COMPARE_REF_TO_TEXTURE),i.texParameteri(w,i.TEXTURE_COMPARE_FUNC,Ze[_.compareFunction])),e.has("EXT_texture_filter_anisotropic")===!0){if(_.magFilter===Mn||_.minFilter!==Da&&_.minFilter!==Fr||_.type===Li&&e.has("OES_texture_float_linear")===!1)return;if(_.anisotropy>1||n.get(_).__currentAnisotropy){const Y=e.get("EXT_texture_filter_anisotropic");i.texParameterf(w,Y.TEXTURE_MAX_ANISOTROPY_EXT,Math.min(_.anisotropy,r.getMaxAnisotropy())),n.get(_).__currentAnisotropy=_.anisotropy}}}function Q(w,_){let Y=!1;w.__webglInit===void 0&&(w.__webglInit=!0,_.addEventListener("dispose",I));const $=_.source;let ee=d.get($);ee===void 0&&(ee={},d.set($,ee));const ge=B(_);if(ge!==w.__cacheKey){ee[ge]===void 0&&(ee[ge]={texture:i.createTexture(),usedTimes:0},a.memory.textures++,Y=!0),ee[ge].usedTimes++;const _e=ee[w.__cacheKey];_e!==void 0&&(ee[w.__cacheKey].usedTimes--,_e.usedTimes===0&&F(_)),w.__cacheKey=ge,w.__webglTexture=ee[ge].texture}return Y}function ue(w,_,Y){return Math.floor(Math.floor(w/Y)/_)}function he(w,_,Y,$){const ge=w.updateRanges;if(ge.length===0)t.texSubImage2D(i.TEXTURE_2D,0,0,0,_.width,_.height,Y,$,_.data);else{ge.sort((Ve,ye)=>Ve.start-ye.start);let _e=0;for(let Ve=1;Ve<ge.length;Ve++){const ye=ge[_e],Se=ge[Ve],ke=ye.start+ye.count,Ye=ue(Se.start,_.width,4),qe=ue(ye.start,_.width,4);Se.start<=ke+1&&Ye===qe&&ue(Se.start+Se.count-1,_.width,4)===Ye?ye.count=Math.max(ye.count,Se.start+Se.count-ye.start):(++_e,ge[_e]=Se)}ge.length=_e+1;const te=t.getParameter(i.UNPACK_ROW_LENGTH),ie=t.getParameter(i.UNPACK_SKIP_PIXELS),xe=t.getParameter(i.UNPACK_SKIP_ROWS);t.pixelStorei(i.UNPACK_ROW_LENGTH,_.width);for(let Ve=0,ye=ge.length;Ve<ye;Ve++){const Se=ge[Ve],ke=Math.floor(Se.start/4),Ye=Math.ceil(Se.count/4),qe=ke%_.width,G=Math.floor(ke/_.width),Me=Ye,oe=1;t.pixelStorei(i.UNPACK_SKIP_PIXELS,qe),t.pixelStorei(i.UNPACK_SKIP_ROWS,G),t.texSubImage2D(i.TEXTURE_2D,0,qe,G,Me,oe,Y,$,_.data)}w.clearUpdateRanges(),t.pixelStorei(i.UNPACK_ROW_LENGTH,te),t.pixelStorei(i.UNPACK_SKIP_PIXELS,ie),t.pixelStorei(i.UNPACK_SKIP_ROWS,xe)}}function Be(w,_,Y){let $=i.TEXTURE_2D;(_.isDataArrayTexture||_.isCompressedArrayTexture)&&($=i.TEXTURE_2D_ARRAY),_.isData3DTexture&&($=i.TEXTURE_3D);const ee=Q(w,_),ge=_.source;t.bindTexture($,w.__webglTexture,i.TEXTURE0+Y);const _e=n.get(ge);if(ge.version!==_e.__version||ee===!0){if(t.activeTexture(i.TEXTURE0+Y),(typeof ImageBitmap<"u"&&_.image instanceof ImageBitmap)===!1){const oe=mt.getPrimaries(mt.workingColorSpace),Ee=_.colorSpace===dr?null:mt.getPrimaries(_.colorSpace),Re=_.colorSpace===dr||oe===Ee?i.NONE:i.BROWSER_DEFAULT_WEBGL;t.pixelStorei(i.UNPACK_FLIP_Y_WEBGL,_.flipY),t.pixelStorei(i.UNPACK_PREMULTIPLY_ALPHA_WEBGL,_.premultiplyAlpha),t.pixelStorei(i.UNPACK_COLORSPACE_CONVERSION_WEBGL,Re)}t.pixelStorei(i.UNPACK_ALIGNMENT,_.unpackAlignment);let ie=m(_.image,!1,r.maxTextureSize);ie=Xe(_,ie);const xe=s.convert(_.format,_.colorSpace),Ve=s.convert(_.type);let ye=S(_.internalFormat,xe,Ve,_.normalized,_.colorSpace,_.isVideoTexture);Ke($,_);let Se;const ke=_.mipmaps,Ye=_.isVideoTexture!==!0,qe=_e.__version===void 0||ee===!0,G=ge.dataReady,Me=y(_,ie);if(_.isDepthTexture)ye=C(_.format===Or,_.type),qe&&(Ye?t.texStorage2D(i.TEXTURE_2D,1,ye,ie.width,ie.height):t.texImage2D(i.TEXTURE_2D,0,ye,ie.width,ie.height,0,xe,Ve,null));else if(_.isDataTexture)if(ke.length>0){Ye&&qe&&t.texStorage2D(i.TEXTURE_2D,Me,ye,ke[0].width,ke[0].height);for(let oe=0,Ee=ke.length;oe<Ee;oe++)Se=ke[oe],Ye?G&&t.texSubImage2D(i.TEXTURE_2D,oe,0,0,Se.width,Se.height,xe,Ve,Se.data):t.texImage2D(i.TEXTURE_2D,oe,ye,Se.width,Se.height,0,xe,Ve,Se.data);_.generateMipmaps=!1}else Ye?(qe&&t.texStorage2D(i.TEXTURE_2D,Me,ye,ie.width,ie.height),G&&he(_,ie,xe,Ve)):t.texImage2D(i.TEXTURE_2D,0,ye,ie.width,ie.height,0,xe,Ve,ie.data);else if(_.isCompressedTexture)if(_.isCompressedArrayTexture){Ye&&qe&&t.texStorage3D(i.TEXTURE_2D_ARRAY,Me,ye,ke[0].width,ke[0].height,ie.depth);for(let oe=0,Ee=ke.length;oe<Ee;oe++)if(Se=ke[oe],_.format!==xi)if(xe!==null)if(Ye){if(G)if(_.layerUpdates.size>0){const Re=mh(Se.width,Se.height,_.format,_.type);for(const de of _.layerUpdates){const Ge=Se.data.subarray(de*Re/Se.data.BYTES_PER_ELEMENT,(de+1)*Re/Se.data.BYTES_PER_ELEMENT);t.compressedTexSubImage3D(i.TEXTURE_2D_ARRAY,oe,0,0,de,Se.width,Se.height,1,xe,Ge)}_.clearLayerUpdates()}else t.compressedTexSubImage3D(i.TEXTURE_2D_ARRAY,oe,0,0,0,Se.width,Se.height,ie.depth,xe,Se.data)}else t.compressedTexImage3D(i.TEXTURE_2D_ARRAY,oe,ye,Se.width,Se.height,ie.depth,0,Se.data,0,0);else Je("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()");else Ye?G&&t.texSubImage3D(i.TEXTURE_2D_ARRAY,oe,0,0,0,Se.width,Se.height,ie.depth,xe,Ve,Se.data):t.texImage3D(i.TEXTURE_2D_ARRAY,oe,ye,Se.width,Se.height,ie.depth,0,xe,Ve,Se.data)}else{Ye&&qe&&t.texStorage2D(i.TEXTURE_2D,Me,ye,ke[0].width,ke[0].height);for(let oe=0,Ee=ke.length;oe<Ee;oe++)Se=ke[oe],_.format!==xi?xe!==null?Ye?G&&t.compressedTexSubImage2D(i.TEXTURE_2D,oe,0,0,Se.width,Se.height,xe,Se.data):t.compressedTexImage2D(i.TEXTURE_2D,oe,ye,Se.width,Se.height,0,Se.data):Je("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()"):Ye?G&&t.texSubImage2D(i.TEXTURE_2D,oe,0,0,Se.width,Se.height,xe,Ve,Se.data):t.texImage2D(i.TEXTURE_2D,oe,ye,Se.width,Se.height,0,xe,Ve,Se.data)}else if(_.isDataArrayTexture)if(Ye){if(qe&&t.texStorage3D(i.TEXTURE_2D_ARRAY,Me,ye,ie.width,ie.height,ie.depth),G)if(_.layerUpdates.size>0){const oe=mh(ie.width,ie.height,_.format,_.type);for(const Ee of _.layerUpdates){const Re=ie.data.subarray(Ee*oe/ie.data.BYTES_PER_ELEMENT,(Ee+1)*oe/ie.data.BYTES_PER_ELEMENT);t.texSubImage3D(i.TEXTURE_2D_ARRAY,0,0,0,Ee,ie.width,ie.height,1,xe,Ve,Re)}_.clearLayerUpdates()}else t.texSubImage3D(i.TEXTURE_2D_ARRAY,0,0,0,0,ie.width,ie.height,ie.depth,xe,Ve,ie.data)}else t.texImage3D(i.TEXTURE_2D_ARRAY,0,ye,ie.width,ie.height,ie.depth,0,xe,Ve,ie.data);else if(_.isData3DTexture)Ye?(qe&&t.texStorage3D(i.TEXTURE_3D,Me,ye,ie.width,ie.height,ie.depth),G&&t.texSubImage3D(i.TEXTURE_3D,0,0,0,0,ie.width,ie.height,ie.depth,xe,Ve,ie.data)):t.texImage3D(i.TEXTURE_3D,0,ye,ie.width,ie.height,ie.depth,0,xe,Ve,ie.data);else if(_.isFramebufferTexture){if(qe)if(Ye)t.texStorage2D(i.TEXTURE_2D,Me,ye,ie.width,ie.height);else{let oe=ie.width,Ee=ie.height;for(let Re=0;Re<Me;Re++)t.texImage2D(i.TEXTURE_2D,Re,ye,oe,Ee,0,xe,Ve,null),oe>>=1,Ee>>=1}}else if(_.isHTMLTexture){if("texElementImage2D"in i){const oe=i.canvas;if(oe.hasAttribute("layoutsubtree")||oe.setAttribute("layoutsubtree","true"),ie.parentNode!==oe){oe.appendChild(ie),h.add(_),oe.onpaint=Ee=>{const Re=Ee.changedElements;for(const de of h)Re.includes(de.image)&&(de.needsUpdate=!0)},oe.requestPaint();return}if(i.texElementImage2D.length===3)i.texElementImage2D(i.TEXTURE_2D,i.RGBA8,ie);else{const Re=i.RGBA,de=i.RGBA,Ge=i.UNSIGNED_BYTE;i.texElementImage2D(i.TEXTURE_2D,0,Re,de,Ge,ie)}i.texParameteri(i.TEXTURE_2D,i.TEXTURE_MIN_FILTER,i.LINEAR),i.texParameteri(i.TEXTURE_2D,i.TEXTURE_WRAP_S,i.CLAMP_TO_EDGE),i.texParameteri(i.TEXTURE_2D,i.TEXTURE_WRAP_T,i.CLAMP_TO_EDGE)}}else if(ke.length>0){if(Ye&&qe){const oe=pe(ke[0]);t.texStorage2D(i.TEXTURE_2D,Me,ye,oe.width,oe.height)}for(let oe=0,Ee=ke.length;oe<Ee;oe++)Se=ke[oe],Ye?G&&t.texSubImage2D(i.TEXTURE_2D,oe,0,0,xe,Ve,Se):t.texImage2D(i.TEXTURE_2D,oe,ye,xe,Ve,Se);_.generateMipmaps=!1}else if(Ye){if(qe){const oe=pe(ie);t.texStorage2D(i.TEXTURE_2D,Me,ye,oe.width,oe.height)}G&&t.texSubImage2D(i.TEXTURE_2D,0,0,0,xe,Ve,ie)}else t.texImage2D(i.TEXTURE_2D,0,ye,xe,Ve,ie);p(_)&&T($),_e.__version=ge.version,_.onUpdate&&_.onUpdate(_)}w.__version=_.version}function Ue(w,_,Y){if(_.image.length!==6)return;const $=Q(w,_),ee=_.source;t.bindTexture(i.TEXTURE_CUBE_MAP,w.__webglTexture,i.TEXTURE0+Y);const ge=n.get(ee);if(ee.version!==ge.__version||$===!0){t.activeTexture(i.TEXTURE0+Y);const _e=mt.getPrimaries(mt.workingColorSpace),te=_.colorSpace===dr?null:mt.getPrimaries(_.colorSpace),ie=_.colorSpace===dr||_e===te?i.NONE:i.BROWSER_DEFAULT_WEBGL;t.pixelStorei(i.UNPACK_FLIP_Y_WEBGL,_.flipY),t.pixelStorei(i.UNPACK_PREMULTIPLY_ALPHA_WEBGL,_.premultiplyAlpha),t.pixelStorei(i.UNPACK_ALIGNMENT,_.unpackAlignment),t.pixelStorei(i.UNPACK_COLORSPACE_CONVERSION_WEBGL,ie);const xe=_.isCompressedTexture||_.image[0].isCompressedTexture,Ve=_.image[0]&&_.image[0].isDataTexture,ye=[];for(let de=0;de<6;de++)!xe&&!Ve?ye[de]=m(_.image[de],!0,r.maxCubemapSize):ye[de]=Ve?_.image[de].image:_.image[de],ye[de]=Xe(_,ye[de]);const Se=ye[0],ke=s.convert(_.format,_.colorSpace),Ye=s.convert(_.type),qe=S(_.internalFormat,ke,Ye,_.normalized,_.colorSpace),G=_.isVideoTexture!==!0,Me=ge.__version===void 0||$===!0,oe=ee.dataReady;let Ee=y(_,Se);Ke(i.TEXTURE_CUBE_MAP,_);let Re;if(xe){G&&Me&&t.texStorage2D(i.TEXTURE_CUBE_MAP,Ee,qe,Se.width,Se.height);for(let de=0;de<6;de++){Re=ye[de].mipmaps;for(let Ge=0;Ge<Re.length;Ge++){const Ne=Re[Ge];_.format!==xi?ke!==null?G?oe&&t.compressedTexSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,Ge,0,0,Ne.width,Ne.height,ke,Ne.data):t.compressedTexImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,Ge,qe,Ne.width,Ne.height,0,Ne.data):Je("WebGLRenderer: Attempt to load unsupported compressed texture format in .setTextureCube()"):G?oe&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,Ge,0,0,Ne.width,Ne.height,ke,Ye,Ne.data):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,Ge,qe,Ne.width,Ne.height,0,ke,Ye,Ne.data)}}}else{if(Re=_.mipmaps,G&&Me){Re.length>0&&Ee++;const de=pe(ye[0]);t.texStorage2D(i.TEXTURE_CUBE_MAP,Ee,qe,de.width,de.height)}for(let de=0;de<6;de++)if(Ve){G?oe&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,0,0,0,ye[de].width,ye[de].height,ke,Ye,ye[de].data):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,0,qe,ye[de].width,ye[de].height,0,ke,Ye,ye[de].data);for(let Ge=0;Ge<Re.length;Ge++){const at=Re[Ge].image[de].image;G?oe&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,Ge+1,0,0,at.width,at.height,ke,Ye,at.data):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,Ge+1,qe,at.width,at.height,0,ke,Ye,at.data)}}else{G?oe&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,0,0,0,ke,Ye,ye[de]):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,0,qe,ke,Ye,ye[de]);for(let Ge=0;Ge<Re.length;Ge++){const Ne=Re[Ge];G?oe&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,Ge+1,0,0,ke,Ye,Ne.image[de]):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+de,Ge+1,qe,ke,Ye,Ne.image[de])}}}p(_)&&T(i.TEXTURE_CUBE_MAP),ge.__version=ee.version,_.onUpdate&&_.onUpdate(_)}w.__version=_.version}function ze(w,_,Y,$,ee,ge){const _e=s.convert(Y.format,Y.colorSpace),te=s.convert(Y.type),ie=S(Y.internalFormat,_e,te,Y.normalized,Y.colorSpace),xe=n.get(_),Ve=n.get(Y);if(Ve.__renderTarget=_,!xe.__hasExternalTextures){const ye=Math.max(1,_.width>>ge),Se=Math.max(1,_.height>>ge);ee===i.TEXTURE_3D||ee===i.TEXTURE_2D_ARRAY?t.texImage3D(ee,ge,ie,ye,Se,_.depth,0,_e,te,null):t.texImage2D(ee,ge,ie,ye,Se,0,_e,te,null)}t.bindFramebuffer(i.FRAMEBUFFER,w),Tt(_)?o.framebufferTexture2DMultisampleEXT(i.FRAMEBUFFER,$,ee,Ve.__webglTexture,0,wt(_)):(ee===i.TEXTURE_2D||ee>=i.TEXTURE_CUBE_MAP_POSITIVE_X&&ee<=i.TEXTURE_CUBE_MAP_NEGATIVE_Z)&&i.framebufferTexture2D(i.FRAMEBUFFER,$,ee,Ve.__webglTexture,ge),t.bindFramebuffer(i.FRAMEBUFFER,null)}function St(w,_,Y){if(i.bindRenderbuffer(i.RENDERBUFFER,w),_.depthBuffer){const $=_.depthTexture,ee=$&&$.isDepthTexture?$.type:null,ge=C(_.stencilBuffer,ee),_e=_.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT;Tt(_)?o.renderbufferStorageMultisampleEXT(i.RENDERBUFFER,wt(_),ge,_.width,_.height):Y?i.renderbufferStorageMultisample(i.RENDERBUFFER,wt(_),ge,_.width,_.height):i.renderbufferStorage(i.RENDERBUFFER,ge,_.width,_.height),i.framebufferRenderbuffer(i.FRAMEBUFFER,_e,i.RENDERBUFFER,w)}else{const $=_.textures;for(let ee=0;ee<$.length;ee++){const ge=$[ee],_e=s.convert(ge.format,ge.colorSpace),te=s.convert(ge.type),ie=S(ge.internalFormat,_e,te,ge.normalized,ge.colorSpace);Tt(_)?o.renderbufferStorageMultisampleEXT(i.RENDERBUFFER,wt(_),ie,_.width,_.height):Y?i.renderbufferStorageMultisample(i.RENDERBUFFER,wt(_),ie,_.width,_.height):i.renderbufferStorage(i.RENDERBUFFER,ie,_.width,_.height)}}i.bindRenderbuffer(i.RENDERBUFFER,null)}function je(w,_,Y){const $=_.isWebGLCubeRenderTarget===!0;if(t.bindFramebuffer(i.FRAMEBUFFER,w),!(_.depthTexture&&_.depthTexture.isDepthTexture))throw new Error("THREE.WebGLTextures: renderTarget.depthTexture must be an instance of THREE.DepthTexture.");const ee=n.get(_.depthTexture);if(ee.__renderTarget=_,(!ee.__webglTexture||_.depthTexture.image.width!==_.width||_.depthTexture.image.height!==_.height)&&(_.depthTexture.image.width=_.width,_.depthTexture.image.height=_.height,_.depthTexture.needsUpdate=!0),$){if(ee.__webglInit===void 0&&(ee.__webglInit=!0,_.depthTexture.addEventListener("dispose",I)),ee.__webglTexture===void 0){ee.__webglTexture=i.createTexture(),t.bindTexture(i.TEXTURE_CUBE_MAP,ee.__webglTexture),Ke(i.TEXTURE_CUBE_MAP,_.depthTexture);const xe=s.convert(_.depthTexture.format),Ve=s.convert(_.depthTexture.type);let ye;_.depthTexture.format===Ji?ye=i.DEPTH_COMPONENT24:_.depthTexture.format===Or&&(ye=i.DEPTH24_STENCIL8);for(let Se=0;Se<6;Se++)i.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Se,0,ye,_.width,_.height,0,xe,Ve,null)}}else j(_.depthTexture,0);const ge=ee.__webglTexture,_e=wt(_),te=$?i.TEXTURE_CUBE_MAP_POSITIVE_X+Y:i.TEXTURE_2D,ie=_.depthTexture.format===Or?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT;if(_.depthTexture.format===Ji)Tt(_)?o.framebufferTexture2DMultisampleEXT(i.FRAMEBUFFER,ie,te,ge,0,_e):i.framebufferTexture2D(i.FRAMEBUFFER,ie,te,ge,0);else if(_.depthTexture.format===Or)Tt(_)?o.framebufferTexture2DMultisampleEXT(i.FRAMEBUFFER,ie,te,ge,0,_e):i.framebufferTexture2D(i.FRAMEBUFFER,ie,te,ge,0);else throw new Error("THREE.WebGLTextures: Unknown depthTexture format.")}function ht(w){const _=n.get(w),Y=w.isWebGLCubeRenderTarget===!0;if(_.__boundDepthTexture!==w.depthTexture){const $=w.depthTexture;if(_.__depthDisposeCallback&&_.__depthDisposeCallback(),$){const ee=()=>{delete _.__boundDepthTexture,delete _.__depthDisposeCallback,$.removeEventListener("dispose",ee)};$.addEventListener("dispose",ee),_.__depthDisposeCallback=ee}_.__boundDepthTexture=$}if(w.depthTexture&&!_.__autoAllocateDepthBuffer)if(Y)for(let $=0;$<6;$++)je(_.__webglFramebuffer[$],w,$);else{const $=w.texture.mipmaps;$&&$.length>0?je(_.__webglFramebuffer[0],w,0):je(_.__webglFramebuffer,w,0)}else if(Y){_.__webglDepthbuffer=[];for(let $=0;$<6;$++)if(t.bindFramebuffer(i.FRAMEBUFFER,_.__webglFramebuffer[$]),_.__webglDepthbuffer[$]===void 0)_.__webglDepthbuffer[$]=i.createRenderbuffer(),St(_.__webglDepthbuffer[$],w,!1);else{const ee=w.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT,ge=_.__webglDepthbuffer[$];i.bindRenderbuffer(i.RENDERBUFFER,ge),i.framebufferRenderbuffer(i.FRAMEBUFFER,ee,i.RENDERBUFFER,ge)}}else{const $=w.texture.mipmaps;if($&&$.length>0?t.bindFramebuffer(i.FRAMEBUFFER,_.__webglFramebuffer[0]):t.bindFramebuffer(i.FRAMEBUFFER,_.__webglFramebuffer),_.__webglDepthbuffer===void 0)_.__webglDepthbuffer=i.createRenderbuffer(),St(_.__webglDepthbuffer,w,!1);else{const ee=w.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT,ge=_.__webglDepthbuffer;i.bindRenderbuffer(i.RENDERBUFFER,ge),i.framebufferRenderbuffer(i.FRAMEBUFFER,ee,i.RENDERBUFFER,ge)}}t.bindFramebuffer(i.FRAMEBUFFER,null)}function ot(w,_,Y){const $=n.get(w);_!==void 0&&ze($.__webglFramebuffer,w,w.texture,i.COLOR_ATTACHMENT0,i.TEXTURE_2D,0),Y!==void 0&&ht(w)}function rt(w){const _=w.texture,Y=n.get(w),$=n.get(_);w.addEventListener("dispose",v);const ee=w.textures,ge=w.isWebGLCubeRenderTarget===!0,_e=ee.length>1;if(_e||($.__webglTexture===void 0&&($.__webglTexture=i.createTexture()),$.__version=_.version,a.memory.textures++),ge){Y.__webglFramebuffer=[];for(let te=0;te<6;te++)if(_.mipmaps&&_.mipmaps.length>0){Y.__webglFramebuffer[te]=[];for(let ie=0;ie<_.mipmaps.length;ie++)Y.__webglFramebuffer[te][ie]=i.createFramebuffer()}else Y.__webglFramebuffer[te]=i.createFramebuffer()}else{if(_.mipmaps&&_.mipmaps.length>0){Y.__webglFramebuffer=[];for(let te=0;te<_.mipmaps.length;te++)Y.__webglFramebuffer[te]=i.createFramebuffer()}else Y.__webglFramebuffer=i.createFramebuffer();if(_e)for(let te=0,ie=ee.length;te<ie;te++){const xe=n.get(ee[te]);xe.__webglTexture===void 0&&(xe.__webglTexture=i.createTexture(),a.memory.textures++)}if(w.samples>0&&Tt(w)===!1){Y.__webglMultisampledFramebuffer=i.createFramebuffer(),Y.__webglColorRenderbuffer=[],t.bindFramebuffer(i.FRAMEBUFFER,Y.__webglMultisampledFramebuffer);for(let te=0;te<ee.length;te++){const ie=ee[te];Y.__webglColorRenderbuffer[te]=i.createRenderbuffer(),i.bindRenderbuffer(i.RENDERBUFFER,Y.__webglColorRenderbuffer[te]);const xe=s.convert(ie.format,ie.colorSpace),Ve=s.convert(ie.type),ye=S(ie.internalFormat,xe,Ve,ie.normalized,ie.colorSpace,w.isXRRenderTarget===!0),Se=wt(w);i.renderbufferStorageMultisample(i.RENDERBUFFER,Se,ye,w.width,w.height),i.framebufferRenderbuffer(i.FRAMEBUFFER,i.COLOR_ATTACHMENT0+te,i.RENDERBUFFER,Y.__webglColorRenderbuffer[te])}i.bindRenderbuffer(i.RENDERBUFFER,null),w.depthBuffer&&(Y.__webglDepthRenderbuffer=i.createRenderbuffer(),St(Y.__webglDepthRenderbuffer,w,!0)),t.bindFramebuffer(i.FRAMEBUFFER,null)}}if(ge){t.bindTexture(i.TEXTURE_CUBE_MAP,$.__webglTexture),Ke(i.TEXTURE_CUBE_MAP,_);for(let te=0;te<6;te++)if(_.mipmaps&&_.mipmaps.length>0)for(let ie=0;ie<_.mipmaps.length;ie++)ze(Y.__webglFramebuffer[te][ie],w,_,i.COLOR_ATTACHMENT0,i.TEXTURE_CUBE_MAP_POSITIVE_X+te,ie);else ze(Y.__webglFramebuffer[te],w,_,i.COLOR_ATTACHMENT0,i.TEXTURE_CUBE_MAP_POSITIVE_X+te,0);p(_)&&T(i.TEXTURE_CUBE_MAP),t.unbindTexture()}else if(_e){for(let te=0,ie=ee.length;te<ie;te++){const xe=ee[te],Ve=n.get(xe);let ye=i.TEXTURE_2D;(w.isWebGL3DRenderTarget||w.isWebGLArrayRenderTarget)&&(ye=w.isWebGL3DRenderTarget?i.TEXTURE_3D:i.TEXTURE_2D_ARRAY),t.bindTexture(ye,Ve.__webglTexture),Ke(ye,xe),ze(Y.__webglFramebuffer,w,xe,i.COLOR_ATTACHMENT0+te,ye,0),p(xe)&&T(ye)}t.unbindTexture()}else{let te=i.TEXTURE_2D;if((w.isWebGL3DRenderTarget||w.isWebGLArrayRenderTarget)&&(te=w.isWebGL3DRenderTarget?i.TEXTURE_3D:i.TEXTURE_2D_ARRAY),t.bindTexture(te,$.__webglTexture),Ke(te,_),_.mipmaps&&_.mipmaps.length>0)for(let ie=0;ie<_.mipmaps.length;ie++)ze(Y.__webglFramebuffer[ie],w,_,i.COLOR_ATTACHMENT0,te,ie);else ze(Y.__webglFramebuffer,w,_,i.COLOR_ATTACHMENT0,te,0);p(_)&&T(te),t.unbindTexture()}w.depthBuffer&&ht(w)}function Ft(w){const _=w.textures;for(let Y=0,$=_.length;Y<$;Y++){const ee=_[Y];if(p(ee)){const ge=P(w),_e=n.get(ee).__webglTexture;t.bindTexture(ge,_e),T(ge),t.unbindTexture()}}}const Gt=[],Ot=[];function _t(w){if(w.samples>0){if(Tt(w)===!1){const _=w.textures,Y=w.width,$=w.height;let ee=i.COLOR_BUFFER_BIT;const ge=w.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT,_e=n.get(w),te=_.length>1;if(te)for(let xe=0;xe<_.length;xe++)t.bindFramebuffer(i.FRAMEBUFFER,_e.__webglMultisampledFramebuffer),i.framebufferRenderbuffer(i.FRAMEBUFFER,i.COLOR_ATTACHMENT0+xe,i.RENDERBUFFER,null),t.bindFramebuffer(i.FRAMEBUFFER,_e.__webglFramebuffer),i.framebufferTexture2D(i.DRAW_FRAMEBUFFER,i.COLOR_ATTACHMENT0+xe,i.TEXTURE_2D,null,0);t.bindFramebuffer(i.READ_FRAMEBUFFER,_e.__webglMultisampledFramebuffer);const ie=w.texture.mipmaps;ie&&ie.length>0?t.bindFramebuffer(i.DRAW_FRAMEBUFFER,_e.__webglFramebuffer[0]):t.bindFramebuffer(i.DRAW_FRAMEBUFFER,_e.__webglFramebuffer);for(let xe=0;xe<_.length;xe++){if(w.resolveDepthBuffer&&(w.depthBuffer&&(ee|=i.DEPTH_BUFFER_BIT),w.stencilBuffer&&w.resolveStencilBuffer&&(ee|=i.STENCIL_BUFFER_BIT)),te){i.framebufferRenderbuffer(i.READ_FRAMEBUFFER,i.COLOR_ATTACHMENT0,i.RENDERBUFFER,_e.__webglColorRenderbuffer[xe]);const Ve=n.get(_[xe]).__webglTexture;i.framebufferTexture2D(i.DRAW_FRAMEBUFFER,i.COLOR_ATTACHMENT0,i.TEXTURE_2D,Ve,0)}i.blitFramebuffer(0,0,Y,$,0,0,Y,$,ee,i.NEAREST),l===!0&&(Gt.length=0,Ot.length=0,Gt.push(i.COLOR_ATTACHMENT0+xe),w.depthBuffer&&w.resolveDepthBuffer===!1&&(Gt.push(ge),Ot.push(ge),i.invalidateFramebuffer(i.DRAW_FRAMEBUFFER,Ot)),i.invalidateFramebuffer(i.READ_FRAMEBUFFER,Gt))}if(t.bindFramebuffer(i.READ_FRAMEBUFFER,null),t.bindFramebuffer(i.DRAW_FRAMEBUFFER,null),te)for(let xe=0;xe<_.length;xe++){t.bindFramebuffer(i.FRAMEBUFFER,_e.__webglMultisampledFramebuffer),i.framebufferRenderbuffer(i.FRAMEBUFFER,i.COLOR_ATTACHMENT0+xe,i.RENDERBUFFER,_e.__webglColorRenderbuffer[xe]);const Ve=n.get(_[xe]).__webglTexture;t.bindFramebuffer(i.FRAMEBUFFER,_e.__webglFramebuffer),i.framebufferTexture2D(i.DRAW_FRAMEBUFFER,i.COLOR_ATTACHMENT0+xe,i.TEXTURE_2D,Ve,0)}t.bindFramebuffer(i.DRAW_FRAMEBUFFER,_e.__webglMultisampledFramebuffer)}else if(w.depthBuffer&&w.resolveDepthBuffer===!1&&l){const _=w.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT;i.invalidateFramebuffer(i.DRAW_FRAMEBUFFER,[_])}}}function wt(w){return Math.min(r.maxSamples,w.samples)}function Tt(w){const _=n.get(w);return w.samples>0&&e.has("WEBGL_multisampled_render_to_texture")===!0&&_.__useRenderToTexture!==!1}function H(w){const _=a.render.frame;f.get(w)!==_&&(f.set(w,_),w.update())}function Xe(w,_){const Y=w.colorSpace,$=w.format,ee=w.type;return w.isCompressedTexture===!0||w.isVideoTexture===!0||Y!==Mo&&Y!==dr&&(mt.getTransfer(Y)===Ct?($!==xi||ee!==Jn)&&Je("WebGLTextures: sRGB encoded textures have to use RGBAFormat and UnsignedByteType."):gt("WebGLTextures: Unsupported texture color space:",Y)),_}function pe(w){return typeof HTMLImageElement<"u"&&w instanceof HTMLImageElement?(c.width=w.naturalWidth||w.width,c.height=w.naturalHeight||w.height):typeof VideoFrame<"u"&&w instanceof VideoFrame?(c.width=w.displayWidth,c.height=w.displayHeight):(c.width=w.width,c.height=w.height),c}this.allocateTextureUnit=N,this.resetTextureUnits=O,this.getTextureUnits=k,this.setTextureUnits=L,this.setTexture2D=j,this.setTexture2DArray=ne,this.setTexture3D=ae,this.setTextureCube=fe,this.rebindTextures=ot,this.setupRenderTarget=rt,this.updateRenderTargetMipmap=Ft,this.updateMultisampleRenderTarget=_t,this.setupDepthRenderbuffer=ht,this.setupFrameBufferTexture=ze,this.useMultisampledRTT=Tt,this.isReversedDepthBuffer=function(){return t.buffers.depth.getReversed()}}function ty(i,e){function t(n,r=dr){let s;const a=mt.getTransfer(r);if(n===Jn)return i.UNSIGNED_BYTE;if(n===au)return i.UNSIGNED_SHORT_4_4_4_4;if(n===ou)return i.UNSIGNED_SHORT_5_5_5_1;if(n===Vd)return i.UNSIGNED_INT_5_9_9_9_REV;if(n===Gd)return i.UNSIGNED_INT_10F_11F_11F_REV;if(n===zd)return i.BYTE;if(n===kd)return i.SHORT;if(n===_a)return i.UNSIGNED_SHORT;if(n===su)return i.INT;if(n===Fi)return i.UNSIGNED_INT;if(n===Li)return i.FLOAT;if(n===Zi)return i.HALF_FLOAT;if(n===Hd)return i.ALPHA;if(n===Wd)return i.RGB;if(n===xi)return i.RGBA;if(n===Ji)return i.DEPTH_COMPONENT;if(n===Or)return i.DEPTH_STENCIL;if(n===Xd)return i.RED;if(n===lu)return i.RED_INTEGER;if(n===Vr)return i.RG;if(n===cu)return i.RG_INTEGER;if(n===uu)return i.RGBA_INTEGER;if(n===lo||n===co||n===uo||n===fo)if(a===Ct)if(s=e.get("WEBGL_compressed_texture_s3tc_srgb"),s!==null){if(n===lo)return s.COMPRESSED_SRGB_S3TC_DXT1_EXT;if(n===co)return s.COMPRESSED_SRGB_ALPHA_S3TC_DXT1_EXT;if(n===uo)return s.COMPRESSED_SRGB_ALPHA_S3TC_DXT3_EXT;if(n===fo)return s.COMPRESSED_SRGB_ALPHA_S3TC_DXT5_EXT}else return null;else if(s=e.get("WEBGL_compressed_texture_s3tc"),s!==null){if(n===lo)return s.COMPRESSED_RGB_S3TC_DXT1_EXT;if(n===co)return s.COMPRESSED_RGBA_S3TC_DXT1_EXT;if(n===uo)return s.COMPRESSED_RGBA_S3TC_DXT3_EXT;if(n===fo)return s.COMPRESSED_RGBA_S3TC_DXT5_EXT}else return null;if(n===uc||n===fc||n===hc||n===dc)if(s=e.get("WEBGL_compressed_texture_pvrtc"),s!==null){if(n===uc)return s.COMPRESSED_RGB_PVRTC_4BPPV1_IMG;if(n===fc)return s.COMPRESSED_RGB_PVRTC_2BPPV1_IMG;if(n===hc)return s.COMPRESSED_RGBA_PVRTC_4BPPV1_IMG;if(n===dc)return s.COMPRESSED_RGBA_PVRTC_2BPPV1_IMG}else return null;if(n===pc||n===mc||n===gc||n===_c||n===xc||n===vo||n===vc)if(s=e.get("WEBGL_compressed_texture_etc"),s!==null){if(n===pc||n===mc)return a===Ct?s.COMPRESSED_SRGB8_ETC2:s.COMPRESSED_RGB8_ETC2;if(n===gc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ETC2_EAC:s.COMPRESSED_RGBA8_ETC2_EAC;if(n===_c)return s.COMPRESSED_R11_EAC;if(n===xc)return s.COMPRESSED_SIGNED_R11_EAC;if(n===vo)return s.COMPRESSED_RG11_EAC;if(n===vc)return s.COMPRESSED_SIGNED_RG11_EAC}else return null;if(n===Sc||n===Mc||n===yc||n===Ec||n===bc||n===Tc||n===Ac||n===wc||n===Rc||n===Cc||n===Pc||n===Dc||n===Lc||n===Ic)if(s=e.get("WEBGL_compressed_texture_astc"),s!==null){if(n===Sc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR:s.COMPRESSED_RGBA_ASTC_4x4_KHR;if(n===Mc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR:s.COMPRESSED_RGBA_ASTC_5x4_KHR;if(n===yc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR:s.COMPRESSED_RGBA_ASTC_5x5_KHR;if(n===Ec)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR:s.COMPRESSED_RGBA_ASTC_6x5_KHR;if(n===bc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR:s.COMPRESSED_RGBA_ASTC_6x6_KHR;if(n===Tc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR:s.COMPRESSED_RGBA_ASTC_8x5_KHR;if(n===Ac)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR:s.COMPRESSED_RGBA_ASTC_8x6_KHR;if(n===wc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR:s.COMPRESSED_RGBA_ASTC_8x8_KHR;if(n===Rc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR:s.COMPRESSED_RGBA_ASTC_10x5_KHR;if(n===Cc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR:s.COMPRESSED_RGBA_ASTC_10x6_KHR;if(n===Pc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR:s.COMPRESSED_RGBA_ASTC_10x8_KHR;if(n===Dc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR:s.COMPRESSED_RGBA_ASTC_10x10_KHR;if(n===Lc)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR:s.COMPRESSED_RGBA_ASTC_12x10_KHR;if(n===Ic)return a===Ct?s.COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR:s.COMPRESSED_RGBA_ASTC_12x12_KHR}else return null;if(n===Uc||n===Nc||n===Fc)if(s=e.get("EXT_texture_compression_bptc"),s!==null){if(n===Uc)return a===Ct?s.COMPRESSED_SRGB_ALPHA_BPTC_UNORM_EXT:s.COMPRESSED_RGBA_BPTC_UNORM_EXT;if(n===Nc)return s.COMPRESSED_RGB_BPTC_SIGNED_FLOAT_EXT;if(n===Fc)return s.COMPRESSED_RGB_BPTC_UNSIGNED_FLOAT_EXT}else return null;if(n===Oc||n===Bc||n===So||n===zc)if(s=e.get("EXT_texture_compression_rgtc"),s!==null){if(n===Oc)return s.COMPRESSED_RED_RGTC1_EXT;if(n===Bc)return s.COMPRESSED_SIGNED_RED_RGTC1_EXT;if(n===So)return s.COMPRESSED_RED_GREEN_RGTC2_EXT;if(n===zc)return s.COMPRESSED_SIGNED_RED_GREEN_RGTC2_EXT}else return null;return n===xa?i.UNSIGNED_INT_24_8:i[n]!==void 0?i[n]:null}return{convert:t}}const ny=`
void main() {

	gl_Position = vec4( position, 1.0 );

}`,iy=`
uniform sampler2DArray depthColor;
uniform float depthWidth;
uniform float depthHeight;

void main() {

	vec2 coord = vec2( gl_FragCoord.x / depthWidth, gl_FragCoord.y / depthHeight );

	if ( coord.x >= 1.0 ) {

		gl_FragDepth = texture( depthColor, vec3( coord.x - 1.0, coord.y, 1 ) ).r;

	} else {

		gl_FragDepth = texture( depthColor, vec3( coord.x, coord.y, 0 ) ).r;

	}

}`;class ry{constructor(){this.texture=null,this.mesh=null,this.depthNear=0,this.depthFar=0}init(e,t){if(this.texture===null){const n=new np(e.texture);(e.depthNear!==t.depthNear||e.depthFar!==t.depthFar)&&(this.depthNear=e.depthNear,this.depthFar=e.depthFar),this.texture=n}}getMesh(e){if(this.texture!==null&&this.mesh===null){const t=e.cameras[0].viewport,n=new Oi({vertexShader:ny,fragmentShader:iy,uniforms:{depthColor:{value:this.texture},depthWidth:{value:t.z},depthHeight:{value:t.w}}});this.mesh=new vi(new ko(20,20),n)}return this.mesh}reset(){this.texture=null,this.mesh=null}getDepthTexture(){return this.texture}}class sy extends Sr{constructor(e,t){super();const n=this;let r=null,s=1,a=null,o="local-floor",l=1,c=null,f=null,h=null,u=null,d=null,x=null;const E=typeof XRWebGLBinding<"u",m=new ry,p={},T=t.getContextAttributes();let P=null,S=null;const C=[],y=[],I=new $e;let v=null;const A=new oi;A.viewport=new $t;const F=new oi;F.viewport=new $t;const D=[A,F],z=new h_;let O=null,k=null;this.cameraAutoUpdate=!0,this.enabled=!1,this.isPresenting=!1,this.getController=function(Q){let ue=C[Q];return ue===void 0&&(ue=new pl,C[Q]=ue),ue.getTargetRaySpace()},this.getControllerGrip=function(Q){let ue=C[Q];return ue===void 0&&(ue=new pl,C[Q]=ue),ue.getGripSpace()},this.getHand=function(Q){let ue=C[Q];return ue===void 0&&(ue=new pl,C[Q]=ue),ue.getHandSpace()};function L(Q){const ue=y.indexOf(Q.inputSource);if(ue===-1)return;const he=C[ue];he!==void 0&&(he.update(Q.inputSource,Q.frame,c||a),he.dispatchEvent({type:Q.type,data:Q.inputSource}))}function N(){r.removeEventListener("select",L),r.removeEventListener("selectstart",L),r.removeEventListener("selectend",L),r.removeEventListener("squeeze",L),r.removeEventListener("squeezestart",L),r.removeEventListener("squeezeend",L),r.removeEventListener("end",N),r.removeEventListener("inputsourceschange",B);for(let Q=0;Q<C.length;Q++){const ue=y[Q];ue!==null&&(y[Q]=null,C[Q].disconnect(ue))}O=null,k=null,m.reset();for(const Q in p)delete p[Q];e.setRenderTarget(P),d=null,u=null,h=null,r=null,S=null,Ke.stop(),n.isPresenting=!1,e.setPixelRatio(v),e.setSize(I.width,I.height,!1),n.dispatchEvent({type:"sessionend"})}this.setFramebufferScaleFactor=function(Q){s=Q,n.isPresenting===!0&&Je("WebXRManager: Cannot change framebuffer scale while presenting.")},this.setReferenceSpaceType=function(Q){o=Q,n.isPresenting===!0&&Je("WebXRManager: Cannot change reference space type while presenting.")},this.getReferenceSpace=function(){return c||a},this.setReferenceSpace=function(Q){c=Q},this.getBaseLayer=function(){return u!==null?u:d},this.getBinding=function(){return h===null&&E&&(h=new XRWebGLBinding(r,t)),h},this.getFrame=function(){return x},this.getSession=function(){return r},this.setSession=async function(Q){if(r=Q,r!==null){if(P=e.getRenderTarget(),r.addEventListener("select",L),r.addEventListener("selectstart",L),r.addEventListener("selectend",L),r.addEventListener("squeeze",L),r.addEventListener("squeezestart",L),r.addEventListener("squeezeend",L),r.addEventListener("end",N),r.addEventListener("inputsourceschange",B),T.xrCompatible!==!0&&await t.makeXRCompatible(),v=e.getPixelRatio(),e.getSize(I),E&&"createProjectionLayer"in XRWebGLBinding.prototype){let he=null,Be=null,Ue=null;T.depth&&(Ue=T.stencil?t.DEPTH24_STENCIL8:t.DEPTH_COMPONENT24,he=T.stencil?Or:Ji,Be=T.stencil?xa:Fi);const ze={colorFormat:t.RGBA8,depthFormat:Ue,scaleFactor:s};h=this.getBinding(),u=h.createProjectionLayer(ze),r.updateRenderState({layers:[u]}),e.setPixelRatio(1),e.setSize(u.textureWidth,u.textureHeight,!1),S=new Ni(u.textureWidth,u.textureHeight,{format:xi,type:Jn,depthTexture:new Is(u.textureWidth,u.textureHeight,Be,void 0,void 0,void 0,void 0,void 0,void 0,he),stencilBuffer:T.stencil,colorSpace:e.outputColorSpace,samples:T.antialias?4:0,resolveDepthBuffer:u.ignoreDepthValues===!1,resolveStencilBuffer:u.ignoreDepthValues===!1})}else{const he={antialias:T.antialias,alpha:!0,depth:T.depth,stencil:T.stencil,framebufferScaleFactor:s};d=new XRWebGLLayer(r,t,he),r.updateRenderState({baseLayer:d}),e.setPixelRatio(1),e.setSize(d.framebufferWidth,d.framebufferHeight,!1),S=new Ni(d.framebufferWidth,d.framebufferHeight,{format:xi,type:Jn,colorSpace:e.outputColorSpace,stencilBuffer:T.stencil,resolveDepthBuffer:d.ignoreDepthValues===!1,resolveStencilBuffer:d.ignoreDepthValues===!1})}S.isXRRenderTarget=!0,this.setFoveation(l),c=null,a=await r.requestReferenceSpace(o),Ke.setContext(r),Ke.start(),n.isPresenting=!0,n.dispatchEvent({type:"sessionstart"})}},this.getEnvironmentBlendMode=function(){if(r!==null)return r.environmentBlendMode},this.getDepthTexture=function(){return m.getDepthTexture()};function B(Q){for(let ue=0;ue<Q.removed.length;ue++){const he=Q.removed[ue],Be=y.indexOf(he);Be>=0&&(y[Be]=null,C[Be].disconnect(he))}for(let ue=0;ue<Q.added.length;ue++){const he=Q.added[ue];let Be=y.indexOf(he);if(Be===-1){for(let ze=0;ze<C.length;ze++)if(ze>=y.length){y.push(he),Be=ze;break}else if(y[ze]===null){y[ze]=he,Be=ze;break}if(Be===-1)break}const Ue=C[Be];Ue&&Ue.connect(he)}}const j=new X,ne=new X;function ae(Q,ue,he){j.setFromMatrixPosition(ue.matrixWorld),ne.setFromMatrixPosition(he.matrixWorld);const Be=j.distanceTo(ne),Ue=ue.projectionMatrix.elements,ze=he.projectionMatrix.elements,St=Ue[14]/(Ue[10]-1),je=Ue[14]/(Ue[10]+1),ht=(Ue[9]+1)/Ue[5],ot=(Ue[9]-1)/Ue[5],rt=(Ue[8]-1)/Ue[0],Ft=(ze[8]+1)/ze[0],Gt=St*rt,Ot=St*Ft,_t=Be/(-rt+Ft),wt=_t*-rt;if(ue.matrixWorld.decompose(Q.position,Q.quaternion,Q.scale),Q.translateX(wt),Q.translateZ(_t),Q.matrixWorld.compose(Q.position,Q.quaternion,Q.scale),Q.matrixWorldInverse.copy(Q.matrixWorld).invert(),Ue[10]===-1)Q.projectionMatrix.copy(ue.projectionMatrix),Q.projectionMatrixInverse.copy(ue.projectionMatrixInverse);else{const Tt=St+_t,H=je+_t,Xe=Gt-wt,pe=Ot+(Be-wt),w=ht*je/H*Tt,_=ot*je/H*Tt;Q.projectionMatrix.makePerspective(Xe,pe,w,_,Tt,H),Q.projectionMatrixInverse.copy(Q.projectionMatrix).invert()}}function fe(Q,ue){ue===null?Q.matrixWorld.copy(Q.matrix):Q.matrixWorld.multiplyMatrices(ue.matrixWorld,Q.matrix),Q.matrixWorldInverse.copy(Q.matrixWorld).invert()}this.updateCamera=function(Q){if(r===null)return;let ue=Q.near,he=Q.far;m.texture!==null&&(m.depthNear>0&&(ue=m.depthNear),m.depthFar>0&&(he=m.depthFar)),z.near=F.near=A.near=ue,z.far=F.far=A.far=he,(O!==z.near||k!==z.far)&&(r.updateRenderState({depthNear:z.near,depthFar:z.far}),O=z.near,k=z.far),z.layers.mask=Q.layers.mask|6,A.layers.mask=z.layers.mask&-5,F.layers.mask=z.layers.mask&-3;const Be=Q.parent,Ue=z.cameras;fe(z,Be);for(let ze=0;ze<Ue.length;ze++)fe(Ue[ze],Be);Ue.length===2?ae(z,A,F):z.projectionMatrix.copy(A.projectionMatrix),re(Q,z,Be)};function re(Q,ue,he){he===null?Q.matrix.copy(ue.matrixWorld):(Q.matrix.copy(he.matrixWorld),Q.matrix.invert(),Q.matrix.multiply(ue.matrixWorld)),Q.matrix.decompose(Q.position,Q.quaternion,Q.scale),Q.updateMatrixWorld(!0),Q.projectionMatrix.copy(ue.projectionMatrix),Q.projectionMatrixInverse.copy(ue.projectionMatrixInverse),Q.isPerspectiveCamera&&(Q.fov=Vc*2*Math.atan(1/Q.projectionMatrix.elements[5]),Q.zoom=1)}this.getCamera=function(){return z},this.getFoveation=function(){if(!(u===null&&d===null))return l},this.setFoveation=function(Q){l=Q,u!==null&&(u.fixedFoveation=Q),d!==null&&d.fixedFoveation!==void 0&&(d.fixedFoveation=Q)},this.hasDepthSensing=function(){return m.texture!==null},this.getDepthSensingMesh=function(){return m.getMesh(z)},this.getCameraTexture=function(Q){return p[Q]};let le=null;function Ze(Q,ue){if(f=ue.getViewerPose(c||a),x=ue,f!==null){const he=f.views;d!==null&&(e.setRenderTargetFramebuffer(S,d.framebuffer),e.setRenderTarget(S));let Be=!1;he.length!==z.cameras.length&&(z.cameras.length=0,Be=!0);for(let je=0;je<he.length;je++){const ht=he[je];let ot=null;if(d!==null)ot=d.getViewport(ht);else{const Ft=h.getViewSubImage(u,ht);ot=Ft.viewport,je===0&&(e.setRenderTargetTextures(S,Ft.colorTexture,Ft.depthStencilTexture),e.setRenderTarget(S))}let rt=D[je];rt===void 0&&(rt=new oi,rt.layers.enable(je),rt.viewport=new $t,D[je]=rt),rt.matrix.fromArray(ht.transform.matrix),rt.matrix.decompose(rt.position,rt.quaternion,rt.scale),rt.projectionMatrix.fromArray(ht.projectionMatrix),rt.projectionMatrixInverse.copy(rt.projectionMatrix).invert(),rt.viewport.set(ot.x,ot.y,ot.width,ot.height),je===0&&(z.matrix.copy(rt.matrix),z.matrix.decompose(z.position,z.quaternion,z.scale)),Be===!0&&z.cameras.push(rt)}const Ue=r.enabledFeatures;if(Ue&&Ue.includes("depth-sensing")&&r.depthUsage=="gpu-optimized"&&E){h=n.getBinding();const je=h.getDepthInformation(he[0]);je&&je.isValid&&je.texture&&m.init(je,r.renderState)}if(Ue&&Ue.includes("camera-access")&&E){e.state.unbindTexture(),h=n.getBinding();for(let je=0;je<he.length;je++){const ht=he[je].camera;if(ht){let ot=p[ht];ot||(ot=new np,p[ht]=ot);const rt=h.getCameraImage(ht);ot.sourceTexture=rt}}}}for(let he=0;he<C.length;he++){const Be=y[he],Ue=C[he];Be!==null&&Ue!==void 0&&Ue.update(Be,ue,c||a)}le&&le(Q,ue),ue.detectedPlanes&&n.dispatchEvent({type:"planesdetected",data:ue}),x=null}const Ke=new ap;Ke.setAnimationLoop(Ze),this.setAnimationLoop=function(Q){le=Q},this.dispose=function(){}}}const ay=new Xt,dp=new nt;dp.set(-1,0,0,0,1,0,0,0,1);function oy(i,e){function t(m,p){m.matrixAutoUpdate===!0&&m.updateMatrix(),p.value.copy(m.matrix)}function n(m,p){p.color.getRGB(m.fogColor.value,ip(i)),p.isFog?(m.fogNear.value=p.near,m.fogFar.value=p.far):p.isFogExp2&&(m.fogDensity.value=p.density)}function r(m,p,T,P,S){p.isNodeMaterial?p.uniformsNeedUpdate=!1:p.isMeshBasicMaterial?s(m,p):p.isMeshLambertMaterial?(s(m,p),p.envMap&&(m.envMapIntensity.value=p.envMapIntensity)):p.isMeshToonMaterial?(s(m,p),h(m,p)):p.isMeshPhongMaterial?(s(m,p),f(m,p),p.envMap&&(m.envMapIntensity.value=p.envMapIntensity)):p.isMeshStandardMaterial?(s(m,p),u(m,p),p.isMeshPhysicalMaterial&&d(m,p,S)):p.isMeshMatcapMaterial?(s(m,p),x(m,p)):p.isMeshDepthMaterial?s(m,p):p.isMeshDistanceMaterial?(s(m,p),E(m,p)):p.isMeshNormalMaterial?s(m,p):p.isLineBasicMaterial?(a(m,p),p.isLineDashedMaterial&&o(m,p)):p.isPointsMaterial?l(m,p,T,P):p.isSpriteMaterial?c(m,p):p.isShadowMaterial?(m.color.value.copy(p.color),m.opacity.value=p.opacity):p.isShaderMaterial&&(p.uniformsNeedUpdate=!1)}function s(m,p){m.opacity.value=p.opacity,p.color&&m.diffuse.value.copy(p.color),p.emissive&&m.emissive.value.copy(p.emissive).multiplyScalar(p.emissiveIntensity),p.map&&(m.map.value=p.map,t(p.map,m.mapTransform)),p.alphaMap&&(m.alphaMap.value=p.alphaMap,t(p.alphaMap,m.alphaMapTransform)),p.bumpMap&&(m.bumpMap.value=p.bumpMap,t(p.bumpMap,m.bumpMapTransform),m.bumpScale.value=p.bumpScale,p.side===Vn&&(m.bumpScale.value*=-1)),p.normalMap&&(m.normalMap.value=p.normalMap,t(p.normalMap,m.normalMapTransform),m.normalScale.value.copy(p.normalScale),p.side===Vn&&m.normalScale.value.negate()),p.displacementMap&&(m.displacementMap.value=p.displacementMap,t(p.displacementMap,m.displacementMapTransform),m.displacementScale.value=p.displacementScale,m.displacementBias.value=p.displacementBias),p.emissiveMap&&(m.emissiveMap.value=p.emissiveMap,t(p.emissiveMap,m.emissiveMapTransform)),p.specularMap&&(m.specularMap.value=p.specularMap,t(p.specularMap,m.specularMapTransform)),p.alphaTest>0&&(m.alphaTest.value=p.alphaTest);const T=e.get(p),P=T.envMap,S=T.envMapRotation;P&&(m.envMap.value=P,m.envMapRotation.value.setFromMatrix4(ay.makeRotationFromEuler(S)).transpose(),P.isCubeTexture&&P.isRenderTargetTexture===!1&&m.envMapRotation.value.premultiply(dp),m.reflectivity.value=p.reflectivity,m.ior.value=p.ior,m.refractionRatio.value=p.refractionRatio),p.lightMap&&(m.lightMap.value=p.lightMap,m.lightMapIntensity.value=p.lightMapIntensity,t(p.lightMap,m.lightMapTransform)),p.aoMap&&(m.aoMap.value=p.aoMap,m.aoMapIntensity.value=p.aoMapIntensity,t(p.aoMap,m.aoMapTransform))}function a(m,p){m.diffuse.value.copy(p.color),m.opacity.value=p.opacity,p.map&&(m.map.value=p.map,t(p.map,m.mapTransform))}function o(m,p){m.dashSize.value=p.dashSize,m.totalSize.value=p.dashSize+p.gapSize,m.scale.value=p.scale}function l(m,p,T,P){m.diffuse.value.copy(p.color),m.opacity.value=p.opacity,m.size.value=p.size*T,m.scale.value=P*.5,p.map&&(m.map.value=p.map,t(p.map,m.uvTransform)),p.alphaMap&&(m.alphaMap.value=p.alphaMap,t(p.alphaMap,m.alphaMapTransform)),p.alphaTest>0&&(m.alphaTest.value=p.alphaTest)}function c(m,p){m.diffuse.value.copy(p.color),m.opacity.value=p.opacity,m.rotation.value=p.rotation,p.map&&(m.map.value=p.map,t(p.map,m.mapTransform)),p.alphaMap&&(m.alphaMap.value=p.alphaMap,t(p.alphaMap,m.alphaMapTransform)),p.alphaTest>0&&(m.alphaTest.value=p.alphaTest)}function f(m,p){m.specular.value.copy(p.specular),m.shininess.value=Math.max(p.shininess,1e-4)}function h(m,p){p.gradientMap&&(m.gradientMap.value=p.gradientMap)}function u(m,p){m.metalness.value=p.metalness,p.metalnessMap&&(m.metalnessMap.value=p.metalnessMap,t(p.metalnessMap,m.metalnessMapTransform)),m.roughness.value=p.roughness,p.roughnessMap&&(m.roughnessMap.value=p.roughnessMap,t(p.roughnessMap,m.roughnessMapTransform)),p.envMap&&(m.envMapIntensity.value=p.envMapIntensity)}function d(m,p,T){m.ior.value=p.ior,p.sheen>0&&(m.sheenColor.value.copy(p.sheenColor).multiplyScalar(p.sheen),m.sheenRoughness.value=p.sheenRoughness,p.sheenColorMap&&(m.sheenColorMap.value=p.sheenColorMap,t(p.sheenColorMap,m.sheenColorMapTransform)),p.sheenRoughnessMap&&(m.sheenRoughnessMap.value=p.sheenRoughnessMap,t(p.sheenRoughnessMap,m.sheenRoughnessMapTransform))),p.clearcoat>0&&(m.clearcoat.value=p.clearcoat,m.clearcoatRoughness.value=p.clearcoatRoughness,p.clearcoatMap&&(m.clearcoatMap.value=p.clearcoatMap,t(p.clearcoatMap,m.clearcoatMapTransform)),p.clearcoatRoughnessMap&&(m.clearcoatRoughnessMap.value=p.clearcoatRoughnessMap,t(p.clearcoatRoughnessMap,m.clearcoatRoughnessMapTransform)),p.clearcoatNormalMap&&(m.clearcoatNormalMap.value=p.clearcoatNormalMap,t(p.clearcoatNormalMap,m.clearcoatNormalMapTransform),m.clearcoatNormalScale.value.copy(p.clearcoatNormalScale),p.side===Vn&&m.clearcoatNormalScale.value.negate())),p.dispersion>0&&(m.dispersion.value=p.dispersion),p.iridescence>0&&(m.iridescence.value=p.iridescence,m.iridescenceIOR.value=p.iridescenceIOR,m.iridescenceThicknessMinimum.value=p.iridescenceThicknessRange[0],m.iridescenceThicknessMaximum.value=p.iridescenceThicknessRange[1],p.iridescenceMap&&(m.iridescenceMap.value=p.iridescenceMap,t(p.iridescenceMap,m.iridescenceMapTransform)),p.iridescenceThicknessMap&&(m.iridescenceThicknessMap.value=p.iridescenceThicknessMap,t(p.iridescenceThicknessMap,m.iridescenceThicknessMapTransform))),p.transmission>0&&(m.transmission.value=p.transmission,m.transmissionSamplerMap.value=T.texture,m.transmissionSamplerSize.value.set(T.width,T.height),p.transmissionMap&&(m.transmissionMap.value=p.transmissionMap,t(p.transmissionMap,m.transmissionMapTransform)),m.thickness.value=p.thickness,p.thicknessMap&&(m.thicknessMap.value=p.thicknessMap,t(p.thicknessMap,m.thicknessMapTransform)),m.attenuationDistance.value=p.attenuationDistance,m.attenuationColor.value.copy(p.attenuationColor)),p.anisotropy>0&&(m.anisotropyVector.value.set(p.anisotropy*Math.cos(p.anisotropyRotation),p.anisotropy*Math.sin(p.anisotropyRotation)),p.anisotropyMap&&(m.anisotropyMap.value=p.anisotropyMap,t(p.anisotropyMap,m.anisotropyMapTransform))),m.specularIntensity.value=p.specularIntensity,m.specularColor.value.copy(p.specularColor),p.specularColorMap&&(m.specularColorMap.value=p.specularColorMap,t(p.specularColorMap,m.specularColorMapTransform)),p.specularIntensityMap&&(m.specularIntensityMap.value=p.specularIntensityMap,t(p.specularIntensityMap,m.specularIntensityMapTransform))}function x(m,p){p.matcap&&(m.matcap.value=p.matcap)}function E(m,p){const T=e.get(p).light;m.referencePosition.value.setFromMatrixPosition(T.matrixWorld),m.nearDistance.value=T.shadow.camera.near,m.farDistance.value=T.shadow.camera.far}return{refreshFogUniforms:n,refreshMaterialUniforms:r}}function ly(i,e,t,n){let r={},s={},a=[];const o=i.getParameter(i.MAX_UNIFORM_BUFFER_BINDINGS);function l(S,C){const y=C.program;n.uniformBlockBinding(S,y)}function c(S,C){let y=r[S.id];y===void 0&&(m(S),y=f(S),r[S.id]=y,S.addEventListener("dispose",T));const I=C.program;n.updateUBOMapping(S,I);const v=e.render.frame;s[S.id]!==v&&(u(S),s[S.id]=v)}function f(S){const C=h();S.__bindingPointIndex=C;const y=i.createBuffer(),I=S.__size,v=S.usage;return i.bindBuffer(i.UNIFORM_BUFFER,y),i.bufferData(i.UNIFORM_BUFFER,I,v),i.bindBuffer(i.UNIFORM_BUFFER,null),i.bindBufferBase(i.UNIFORM_BUFFER,C,y),y}function h(){for(let S=0;S<o;S++)if(a.indexOf(S)===-1)return a.push(S),S;return gt("WebGLRenderer: Maximum number of simultaneously usable uniforms groups reached."),0}function u(S){const C=r[S.id],y=S.uniforms,I=S.__cache;i.bindBuffer(i.UNIFORM_BUFFER,C);for(let v=0,A=y.length;v<A;v++){const F=y[v];if(Array.isArray(F))for(let D=0,z=F.length;D<z;D++)d(F[D],v,D,I);else d(F,v,0,I)}i.bindBuffer(i.UNIFORM_BUFFER,null)}function d(S,C,y,I){if(E(S,C,y,I)===!0){const v=S.__offset,A=S.value;if(Array.isArray(A)){let F=0;for(let D=0;D<A.length;D++){const z=A[D],O=p(z);x(z,S.__data,F),typeof z!="number"&&typeof z!="boolean"&&!z.isMatrix3&&!ArrayBuffer.isView(z)&&(F+=O.storage/Float32Array.BYTES_PER_ELEMENT)}}else x(A,S.__data,0);i.bufferSubData(i.UNIFORM_BUFFER,v,S.__data)}}function x(S,C,y){typeof S=="number"||typeof S=="boolean"?C[0]=S:S.isMatrix3?(C[0]=S.elements[0],C[1]=S.elements[1],C[2]=S.elements[2],C[3]=0,C[4]=S.elements[3],C[5]=S.elements[4],C[6]=S.elements[5],C[7]=0,C[8]=S.elements[6],C[9]=S.elements[7],C[10]=S.elements[8],C[11]=0):ArrayBuffer.isView(S)?C.set(new S.constructor(S.buffer,S.byteOffset,C.length)):S.toArray(C,y)}function E(S,C,y,I){const v=S.value,A=C+"_"+y;if(I[A]===void 0)return typeof v=="number"||typeof v=="boolean"?I[A]=v:ArrayBuffer.isView(v)?I[A]=v.slice():I[A]=v.clone(),!0;{const F=I[A];if(typeof v=="number"||typeof v=="boolean"){if(F!==v)return I[A]=v,!0}else{if(ArrayBuffer.isView(v))return!0;if(F.equals(v)===!1)return F.copy(v),!0}}return!1}function m(S){const C=S.uniforms;let y=0;const I=16;for(let A=0,F=C.length;A<F;A++){const D=Array.isArray(C[A])?C[A]:[C[A]];for(let z=0,O=D.length;z<O;z++){const k=D[z],L=Array.isArray(k.value)?k.value:[k.value];for(let N=0,B=L.length;N<B;N++){const j=L[N],ne=p(j),ae=y%I,fe=ae%ne.boundary,re=ae+fe;y+=fe,re!==0&&I-re<ne.storage&&(y+=I-re),k.__data=new Float32Array(ne.storage/Float32Array.BYTES_PER_ELEMENT),k.__offset=y,y+=ne.storage}}}const v=y%I;return v>0&&(y+=I-v),S.__size=y,S.__cache={},this}function p(S){const C={boundary:0,storage:0};return typeof S=="number"||typeof S=="boolean"?(C.boundary=4,C.storage=4):S.isVector2?(C.boundary=8,C.storage=8):S.isVector3||S.isColor?(C.boundary=16,C.storage=12):S.isVector4?(C.boundary=16,C.storage=16):S.isMatrix3?(C.boundary=48,C.storage=48):S.isMatrix4?(C.boundary=64,C.storage=64):S.isTexture?Je("WebGLRenderer: Texture samplers can not be part of an uniforms group."):ArrayBuffer.isView(S)?(C.boundary=16,C.storage=S.byteLength):Je("WebGLRenderer: Unsupported uniform value type.",S),C}function T(S){const C=S.target;C.removeEventListener("dispose",T);const y=a.indexOf(C.__bindingPointIndex);a.splice(y,1),i.deleteBuffer(r[C.id]),delete r[C.id],delete s[C.id]}function P(){for(const S in r)i.deleteBuffer(r[S]);a=[],r={},s={}}return{bind:l,update:c,dispose:P}}const cy=new Uint16Array([12469,15057,12620,14925,13266,14620,13807,14376,14323,13990,14545,13625,14713,13328,14840,12882,14931,12528,14996,12233,15039,11829,15066,11525,15080,11295,15085,10976,15082,10705,15073,10495,13880,14564,13898,14542,13977,14430,14158,14124,14393,13732,14556,13410,14702,12996,14814,12596,14891,12291,14937,11834,14957,11489,14958,11194,14943,10803,14921,10506,14893,10278,14858,9960,14484,14039,14487,14025,14499,13941,14524,13740,14574,13468,14654,13106,14743,12678,14818,12344,14867,11893,14889,11509,14893,11180,14881,10751,14852,10428,14812,10128,14765,9754,14712,9466,14764,13480,14764,13475,14766,13440,14766,13347,14769,13070,14786,12713,14816,12387,14844,11957,14860,11549,14868,11215,14855,10751,14825,10403,14782,10044,14729,9651,14666,9352,14599,9029,14967,12835,14966,12831,14963,12804,14954,12723,14936,12564,14917,12347,14900,11958,14886,11569,14878,11247,14859,10765,14828,10401,14784,10011,14727,9600,14660,9289,14586,8893,14508,8533,15111,12234,15110,12234,15104,12216,15092,12156,15067,12010,15028,11776,14981,11500,14942,11205,14902,10752,14861,10393,14812,9991,14752,9570,14682,9252,14603,8808,14519,8445,14431,8145,15209,11449,15208,11451,15202,11451,15190,11438,15163,11384,15117,11274,15055,10979,14994,10648,14932,10343,14871,9936,14803,9532,14729,9218,14645,8742,14556,8381,14461,8020,14365,7603,15273,10603,15272,10607,15267,10619,15256,10631,15231,10614,15182,10535,15118,10389,15042,10167,14963,9787,14883,9447,14800,9115,14710,8665,14615,8318,14514,7911,14411,7507,14279,7198,15314,9675,15313,9683,15309,9712,15298,9759,15277,9797,15229,9773,15166,9668,15084,9487,14995,9274,14898,8910,14800,8539,14697,8234,14590,7790,14479,7409,14367,7067,14178,6621,15337,8619,15337,8631,15333,8677,15325,8769,15305,8871,15264,8940,15202,8909,15119,8775,15022,8565,14916,8328,14804,8009,14688,7614,14569,7287,14448,6888,14321,6483,14088,6171,15350,7402,15350,7419,15347,7480,15340,7613,15322,7804,15287,7973,15229,8057,15148,8012,15046,7846,14933,7611,14810,7357,14682,7069,14552,6656,14421,6316,14251,5948,14007,5528,15356,5942,15356,5977,15353,6119,15348,6294,15332,6551,15302,6824,15249,7044,15171,7122,15070,7050,14949,6861,14818,6611,14679,6349,14538,6067,14398,5651,14189,5311,13935,4958,15359,4123,15359,4153,15356,4296,15353,4646,15338,5160,15311,5508,15263,5829,15188,6042,15088,6094,14966,6001,14826,5796,14678,5543,14527,5287,14377,4985,14133,4586,13869,4257,15360,1563,15360,1642,15358,2076,15354,2636,15341,3350,15317,4019,15273,4429,15203,4732,15105,4911,14981,4932,14836,4818,14679,4621,14517,4386,14359,4156,14083,3795,13808,3437,15360,122,15360,137,15358,285,15355,636,15344,1274,15322,2177,15281,2765,15215,3223,15120,3451,14995,3569,14846,3567,14681,3466,14511,3305,14344,3121,14037,2800,13753,2467,15360,0,15360,1,15359,21,15355,89,15346,253,15325,479,15287,796,15225,1148,15133,1492,15008,1749,14856,1882,14685,1886,14506,1783,14324,1608,13996,1398,13702,1183]);let wi=null;function uy(){return wi===null&&(wi=new $0(cy,16,16,Vr,Zi),wi.name="DFG_LUT",wi.minFilter=yn,wi.magFilter=yn,wi.wrapS=Xi,wi.wrapT=Xi,wi.generateMipmaps=!1,wi.needsUpdate=!0),wi}class fy{constructor(e={}){const{canvas:t=E0(),context:n=null,depth:r=!0,stencil:s=!1,alpha:a=!1,antialias:o=!1,premultipliedAlpha:l=!0,preserveDrawingBuffer:c=!1,powerPreference:f="default",failIfMajorPerformanceCaveat:h=!1,reversedDepthBuffer:u=!1,outputBufferType:d=Jn}=e;this.isWebGLRenderer=!0;let x;if(n!==null){if(typeof WebGLRenderingContext<"u"&&n instanceof WebGLRenderingContext)throw new Error("THREE.WebGLRenderer: WebGL 1 is not supported since r163.");x=n.getContextAttributes().alpha}else x=a;const E=d,m=new Set([uu,cu,lu]),p=new Set([Jn,Fi,_a,xa,au,ou]),T=new Uint32Array(4),P=new Int32Array(4),S=new X;let C=null,y=null;const I=[],v=[];let A=null;this.domElement=t,this.debug={checkShaderErrors:!0,onShaderError:null},this.autoClear=!0,this.autoClearColor=!0,this.autoClearDepth=!0,this.autoClearStencil=!0,this.sortObjects=!0,this.clippingPlanes=[],this.localClippingEnabled=!1,this.toneMapping=Ui,this.toneMappingExposure=1,this.transmissionResolutionScale=1;const F=this;let D=!1,z=null,O=null,k=null,L=null;this._outputColorSpace=ai;let N=0,B=0,j=null,ne=-1,ae=null;const fe=new $t,re=new $t;let le=null;const Ze=new ft(0);let Ke=0,Q=t.width,ue=t.height,he=1,Be=null,Ue=null;const ze=new $t(0,0,Q,ue),St=new $t(0,0,Q,ue);let je=!1;const ht=new gu;let ot=!1,rt=!1;const Ft=new Xt,Gt=new X,Ot=new $t,_t={background:null,fog:null,environment:null,overrideMaterial:null,isScene:!0};let wt=!1;function Tt(){return j===null?he:1}let H=n;function Xe(b,W){return t.getContext(b,W)}try{const b={alpha:!0,depth:r,stencil:s,antialias:o,premultipliedAlpha:l,preserveDrawingBuffer:c,powerPreference:f,failIfMajorPerformanceCaveat:h};if("setAttribute"in t&&t.setAttribute("data-engine",`three.js r${ru}`),t.addEventListener("webglcontextlost",at,!1),t.addEventListener("webglcontextrestored",lt,!1),t.addEventListener("webglcontextcreationerror",xn,!1),H===null){const W="webgl2";if(H=Xe(W,b),H===null)throw Xe(W)?new Error("THREE.WebGLRenderer: Error creating WebGL context with your selected attributes."):new Error("THREE.WebGLRenderer: Error creating WebGL context.")}}catch(b){throw gt("WebGLRenderer: "+b.message),b}let pe,w,_,Y,$,ee,ge,_e,te,ie,xe,Ve,ye,Se,ke,Ye,qe,G,Me,oe,Ee,Re,de;function Ge(){pe=new uS(H),pe.init(),Ee=new ty(H,pe),w=new nS(H,pe,e,Ee),_=new QM(H,pe),w.reversedDepthBuffer&&u&&_.buffers.depth.setReversed(!0),O=H.createFramebuffer(),k=H.createFramebuffer(),L=H.createFramebuffer(),Y=new dS(H),$=new zM,ee=new ey(H,pe,_,$,w,Ee,Y),ge=new cS(F),_e=new __(H),Re=new eS(H,_e),te=new fS(H,_e,Y,Re),ie=new mS(H,te,_e,Re,Y),G=new pS(H,w,ee),ke=new iS($),xe=new BM(F,ge,pe,w,Re,ke),Ve=new oy(F,$),ye=new VM,Se=new qM(pe),qe=new Qv(F,ge,_,ie,x,l),Ye=new jM(F,ie,w),de=new ly(H,Y,w,_),Me=new tS(H,pe,Y),oe=new hS(H,pe,Y),Y.programs=xe.programs,F.capabilities=w,F.extensions=pe,F.properties=$,F.renderLists=ye,F.shadowMap=Ye,F.state=_,F.info=Y}Ge(),E!==Jn&&(A=new _S(E,t.width,t.height,o,r,s));const Ne=new sy(F,H);this.xr=Ne,this.getContext=function(){return H},this.getContextAttributes=function(){return H.getContextAttributes()},this.forceContextLoss=function(){const b=pe.get("WEBGL_lose_context");b&&b.loseContext()},this.forceContextRestore=function(){const b=pe.get("WEBGL_lose_context");b&&b.restoreContext()},this.getPixelRatio=function(){return he},this.setPixelRatio=function(b){b!==void 0&&(he=b,this.setSize(Q,ue,!1))},this.getSize=function(b){return b.set(Q,ue)},this.setSize=function(b,W,J=!0){if(Ne.isPresenting){Je("WebGLRenderer: Can't change size while VR device is presenting.");return}Q=b,ue=W,t.width=Math.floor(b*he),t.height=Math.floor(W*he),J===!0&&(t.style.width=b+"px",t.style.height=W+"px"),A!==null&&A.setSize(t.width,t.height),this.setViewport(0,0,b,W)},this.getDrawingBufferSize=function(b){return b.set(Q*he,ue*he).floor()},this.setDrawingBufferSize=function(b,W,J){Q=b,ue=W,he=J,t.width=Math.floor(b*J),t.height=Math.floor(W*J),this.setViewport(0,0,b,W)},this.setEffects=function(b){if(E===Jn){gt("WebGLRenderer: setEffects() requires outputBufferType set to HalfFloatType or FloatType.");return}if(b){for(let W=0;W<b.length;W++)if(b[W].isOutputPass===!0){Je("WebGLRenderer: OutputPass is not needed in setEffects(). Tone mapping and color space conversion are applied automatically.");break}}A.setEffects(b||[])},this.getCurrentViewport=function(b){return b.copy(fe)},this.getViewport=function(b){return b.copy(ze)},this.setViewport=function(b,W,J,q){b.isVector4?ze.set(b.x,b.y,b.z,b.w):ze.set(b,W,J,q),_.viewport(fe.copy(ze).multiplyScalar(he).round())},this.getScissor=function(b){return b.copy(St)},this.setScissor=function(b,W,J,q){b.isVector4?St.set(b.x,b.y,b.z,b.w):St.set(b,W,J,q),_.scissor(re.copy(St).multiplyScalar(he).round())},this.getScissorTest=function(){return je},this.setScissorTest=function(b){_.setScissorTest(je=b)},this.setOpaqueSort=function(b){Be=b},this.setTransparentSort=function(b){Ue=b},this.getClearColor=function(b){return b.copy(qe.getClearColor())},this.setClearColor=function(){qe.setClearColor(...arguments)},this.getClearAlpha=function(){return qe.getClearAlpha()},this.setClearAlpha=function(){qe.setClearAlpha(...arguments)},this.clear=function(b=!0,W=!0,J=!0){let q=0;if(b){let Z=!1;if(j!==null){const Te=j.texture.format;Z=m.has(Te)}if(Z){const Te=j.texture.type,De=p.has(Te),be=qe.getClearColor(),Oe=qe.getClearAlpha(),Ce=be.r,et=be.g,it=be.b;De?(T[0]=Ce,T[1]=et,T[2]=it,T[3]=Oe,H.clearBufferuiv(H.COLOR,0,T)):(P[0]=Ce,P[1]=et,P[2]=it,P[3]=Oe,H.clearBufferiv(H.COLOR,0,P))}else q|=H.COLOR_BUFFER_BIT}W&&(q|=H.DEPTH_BUFFER_BIT,this.state.buffers.depth.setMask(!0)),J&&(q|=H.STENCIL_BUFFER_BIT,this.state.buffers.stencil.setMask(4294967295)),q!==0&&H.clear(q)},this.clearColor=function(){this.clear(!0,!1,!1)},this.clearDepth=function(){this.clear(!1,!0,!1)},this.clearStencil=function(){this.clear(!1,!1,!0)},this.setNodesHandler=function(b){b.setRenderer(this),z=b},this.dispose=function(){t.removeEventListener("webglcontextlost",at,!1),t.removeEventListener("webglcontextrestored",lt,!1),t.removeEventListener("webglcontextcreationerror",xn,!1),qe.dispose(),ye.dispose(),Se.dispose(),$.dispose(),ge.dispose(),ie.dispose(),Re.dispose(),de.dispose(),xe.dispose(),Ne.dispose(),Ne.removeEventListener("sessionstart",Ea),Ne.removeEventListener("sessionend",ba),Rn.stop()};function at(b){b.preventDefault(),bo("WebGLRenderer: Context Lost."),D=!0}function lt(){bo("WebGLRenderer: Context Restored."),D=!1;const b=Y.autoReset,W=Ye.enabled,J=Ye.autoUpdate,q=Ye.needsUpdate,Z=Ye.type;Ge(),Y.autoReset=b,Ye.enabled=W,Ye.autoUpdate=J,Ye.needsUpdate=q,Ye.type=Z}function xn(b){gt("WebGLRenderer: A WebGL context could not be created. Reason: ",b.statusMessage)}function vn(b){const W=b.target;W.removeEventListener("dispose",vn),zn(W)}function zn(b){ji(b),$.remove(b)}function ji(b){const W=$.get(b).programs;W!==void 0&&(W.forEach(function(J){xe.releaseProgram(J)}),b.isShaderMaterial&&xe.releaseShaderCache(b))}this.renderBufferDirect=function(b,W,J,q,Z,Te){W===null&&(W=_t);const De=Z.isMesh&&Z.matrixWorld.determinantAffine()<0,be=yr(b,W,J,q,Z);_.setMaterial(q,De);let Oe=J.index,Ce=1;if(q.wireframe===!0){if(Oe=te.getWireframeAttribute(J),Oe===void 0)return;Ce=2}const et=J.drawRange,it=J.attributes.position;let He=et.start*Ce,Mt=(et.start+et.count)*Ce;Te!==null&&(He=Math.max(He,Te.start*Ce),Mt=Math.min(Mt,(Te.start+Te.count)*Ce)),Oe!==null?(He=Math.max(He,0),Mt=Math.min(Mt,Oe.count)):it!=null&&(He=Math.max(He,0),Mt=Math.min(Mt,it.count));const qt=Mt-He;if(qt<0||qt===1/0)return;Re.setup(Z,q,be,J,Oe);let Ht,Rt=Me;if(Oe!==null&&(Ht=_e.get(Oe),Rt=oe,Rt.setIndex(Ht)),Z.isMesh)q.wireframe===!0?(_.setLineWidth(q.wireframeLinewidth*Tt()),Rt.setMode(H.LINES)):Rt.setMode(H.TRIANGLES);else if(Z.isLine){let rn=q.linewidth;rn===void 0&&(rn=1),_.setLineWidth(rn*Tt()),Z.isLineSegments?Rt.setMode(H.LINES):Z.isLineLoop?Rt.setMode(H.LINE_LOOP):Rt.setMode(H.LINE_STRIP)}else Z.isPoints?Rt.setMode(H.POINTS):Z.isSprite&&Rt.setMode(H.TRIANGLES);if(Z.isBatchedMesh)if(pe.get("WEBGL_multi_draw"))Rt.renderMultiDraw(Z._multiDrawStarts,Z._multiDrawCounts,Z._multiDrawCount);else{const rn=Z._multiDrawStarts,Le=Z._multiDrawCounts,Cn=Z._multiDrawCount,dt=Oe?_e.get(Oe).bytesPerElement:1,En=$.get(q).currentProgram.getUniforms();for(let Pn=0;Pn<Cn;Pn++)En.setValue(H,"_gl_DrawID",Pn),Rt.render(rn[Pn]/dt,Le[Pn])}else if(Z.isInstancedMesh)Rt.renderInstances(He,qt,Z.count);else if(J.isInstancedBufferGeometry){const rn=J._maxInstanceCount!==void 0?J._maxInstanceCount:1/0,Le=Math.min(J.instanceCount,rn);Rt.renderInstances(He,qt,Le)}else Rt.render(He,qt)};function Bi(b,W,J){b.transparent===!0&&b.side===Pi&&b.forceSinglePass===!1?(b.side=Vn,b.needsUpdate=!0,Yt(b,W,J),b.side=xr,b.needsUpdate=!0,Yt(b,W,J),b.side=Pi):Yt(b,W,J)}this.compile=function(b,W,J=null){J===null&&(J=b),y=Se.get(J),y.init(W),v.push(y),J.traverseVisible(function(Z){Z.isLight&&Z.layers.test(W.layers)&&(y.pushLight(Z),Z.castShadow&&y.pushShadow(Z))}),b!==J&&b.traverseVisible(function(Z){Z.isLight&&Z.layers.test(W.layers)&&(y.pushLight(Z),Z.castShadow&&y.pushShadow(Z))}),y.setupLights();const q=new Set;return b.traverse(function(Z){if(!(Z.isMesh||Z.isPoints||Z.isLine||Z.isSprite))return;const Te=Z.material;if(Te)if(Array.isArray(Te))for(let De=0;De<Te.length;De++){const be=Te[De];Bi(be,J,Z),q.add(be)}else Bi(Te,J,Z),q.add(Te)}),y=v.pop(),q},this.compileAsync=function(b,W,J=null){const q=this.compile(b,W,J);return new Promise(Z=>{function Te(){if(q.forEach(function(De){$.get(De).currentProgram.isReady()&&q.delete(De)}),q.size===0){Z(b);return}setTimeout(Te,10)}pe.get("KHR_parallel_shader_compile")!==null?Te():setTimeout(Te,10)})};let Wr=null;function ya(b){Wr&&Wr(b)}function Ea(){Rn.stop()}function ba(){Rn.start()}const Rn=new ap;Rn.setAnimationLoop(ya),typeof self<"u"&&Rn.setContext(self),this.setAnimationLoop=function(b){Wr=b,Ne.setAnimationLoop(b),b===null?Rn.stop():Rn.start()},Ne.addEventListener("sessionstart",Ea),Ne.addEventListener("sessionend",ba),this.render=function(b,W){if(W!==void 0&&W.isCamera!==!0){gt("WebGLRenderer.render: camera is not an instance of THREE.Camera.");return}if(D===!0)return;z!==null&&z.renderStart(b,W);const J=Ne.enabled===!0&&Ne.isPresenting===!0,q=A!==null&&(j===null||J)&&A.begin(F,j);if(b.matrixWorldAutoUpdate===!0&&b.updateMatrixWorld(),W.parent===null&&W.matrixWorldAutoUpdate===!0&&W.updateMatrixWorld(),Ne.enabled===!0&&Ne.isPresenting===!0&&(A===null||A.isCompositing()===!1)&&(Ne.cameraAutoUpdate===!0&&Ne.updateCamera(W),W=Ne.getCamera()),b.isScene===!0&&b.onBeforeRender(F,b,W,j),y=Se.get(b,v.length),y.init(W),y.state.textureUnits=ee.getTextureUnits(),v.push(y),Ft.multiplyMatrices(W.projectionMatrix,W.matrixWorldInverse),ht.setFromProjectionMatrix(Ft,Ii,W.reversedDepth),rt=this.localClippingEnabled,ot=ke.init(this.clippingPlanes,rt),C=ye.get(b,I.length),C.init(),I.push(C),Ne.enabled===!0&&Ne.isPresenting===!0){const De=F.xr.getDepthSensingMesh();De!==null&&ks(De,W,-1/0,F.sortObjects)}ks(b,W,0,F.sortObjects),C.finish(),F.sortObjects===!0&&C.sort(Be,Ue,W.reversedDepth),wt=Ne.enabled===!1||Ne.isPresenting===!1||Ne.hasDepthSensing()===!1,wt&&qe.addToRenderList(C,b),this.info.render.frame++,this.info.autoReset===!0&&this.info.reset(),ot===!0&&ke.beginShadows();const Z=y.state.shadowsArray;if(Ye.render(Z,b,W),ot===!0&&ke.endShadows(),(q&&A.hasRenderPass())===!1){const De=C.opaque,be=C.transmissive;if(y.setupLights(),W.isArrayCamera){const Oe=W.cameras;if(be.length>0)for(let Ce=0,et=Oe.length;Ce<et;Ce++){const it=Oe[Ce];Vs(De,be,b,it)}wt&&qe.render(b);for(let Ce=0,et=Oe.length;Ce<et;Ce++){const it=Oe[Ce];Qi(C,b,it,it.viewport)}}else be.length>0&&Vs(De,be,b,W),wt&&qe.render(b),Qi(C,b,W)}j!==null&&B===0&&(ee.updateMultisampleRenderTarget(j),ee.updateRenderTargetMipmap(j)),q&&A.end(F),b.isScene===!0&&b.onAfterRender(F,b,W),Re.resetDefaultState(),ne=-1,ae=null,v.pop(),v.length>0?(y=v[v.length-1],ee.setTextureUnits(y.state.textureUnits),ot===!0&&ke.setGlobalState(F.clippingPlanes,y.state.camera)):y=null,I.pop(),I.length>0?C=I[I.length-1]:C=null,z!==null&&z.renderEnd()};function ks(b,W,J,q){if(b.visible===!1)return;if(b.layers.test(W.layers)){if(b.isGroup)J=b.renderOrder;else if(b.isLOD)b.autoUpdate===!0&&b.update(W);else if(b.isLightProbeGrid)y.pushLightProbeGrid(b);else if(b.isLight)y.pushLight(b),b.castShadow&&y.pushShadow(b);else if(b.isSprite){if(!b.frustumCulled||ht.intersectsSprite(b)){q&&Ot.setFromMatrixPosition(b.matrixWorld).applyMatrix4(Ft);const De=ie.update(b),be=b.material;be.visible&&C.push(b,De,be,J,Ot.z,null)}}else if((b.isMesh||b.isLine||b.isPoints)&&(!b.frustumCulled||ht.intersectsObject(b))){const De=ie.update(b),be=b.material;if(q&&(b.boundingSphere!==void 0?(b.boundingSphere===null&&b.computeBoundingSphere(),Ot.copy(b.boundingSphere.center)):(De.boundingSphere===null&&De.computeBoundingSphere(),Ot.copy(De.boundingSphere.center)),Ot.applyMatrix4(b.matrixWorld).applyMatrix4(Ft)),Array.isArray(be)){const Oe=De.groups;for(let Ce=0,et=Oe.length;Ce<et;Ce++){const it=Oe[Ce],He=be[it.materialIndex];He&&He.visible&&C.push(b,De,He,J,Ot.z,it)}}else be.visible&&C.push(b,De,be,J,Ot.z,null)}}const Te=b.children;for(let De=0,be=Te.length;De<be;De++)ks(Te[De],W,J,q)}function Qi(b,W,J,q){const{opaque:Z,transmissive:Te,transparent:De}=b;y.setupLightsView(J),ot===!0&&ke.setGlobalState(F.clippingPlanes,J),q&&_.viewport(fe.copy(q)),Z.length>0&&Gn(Z,W,J),Te.length>0&&Gn(Te,W,J),De.length>0&&Gn(De,W,J),_.buffers.depth.setTest(!0),_.buffers.depth.setMask(!0),_.buffers.color.setMask(!0),_.setPolygonOffset(!1)}function Vs(b,W,J,q){if((J.isScene===!0?J.overrideMaterial:null)!==null)return;if(y.state.transmissionRenderTarget[q.id]===void 0){const He=pe.has("EXT_color_buffer_half_float")||pe.has("EXT_color_buffer_float");y.state.transmissionRenderTarget[q.id]=new Ni(1,1,{generateMipmaps:!0,type:He?Zi:Jn,minFilter:Fr,samples:Math.max(4,w.samples),stencilBuffer:s,resolveDepthBuffer:!1,resolveStencilBuffer:!1,colorSpace:mt.workingColorSpace})}const Te=y.state.transmissionRenderTarget[q.id],De=q.viewport||fe;Te.setSize(De.z*F.transmissionResolutionScale,De.w*F.transmissionResolutionScale);const be=F.getRenderTarget(),Oe=F.getActiveCubeFace(),Ce=F.getActiveMipmapLevel();F.setRenderTarget(Te),F.getClearColor(Ze),Ke=F.getClearAlpha(),Ke<1&&F.setClearColor(16777215,.5),F.clear(),wt&&qe.render(J);const et=F.toneMapping;F.toneMapping=Ui;const it=q.viewport;if(q.viewport!==void 0&&(q.viewport=void 0),y.setupLightsView(q),ot===!0&&ke.setGlobalState(F.clippingPlanes,q),Gn(b,J,q),ee.updateMultisampleRenderTarget(Te),ee.updateRenderTargetMipmap(Te),pe.has("WEBGL_multisampled_render_to_texture")===!1){let He=!1;for(let Mt=0,qt=W.length;Mt<qt;Mt++){const Ht=W[Mt],{object:Rt,geometry:rn,material:Le,group:Cn}=Ht;if(Le.side===Pi&&Rt.layers.test(q.layers)){const dt=Le.side;Le.side=Vn,Le.needsUpdate=!0,Qt(Rt,J,q,rn,Le,Cn),Le.side=dt,Le.needsUpdate=!0,He=!0}}He===!0&&(ee.updateMultisampleRenderTarget(Te),ee.updateRenderTargetMipmap(Te))}F.setRenderTarget(be,Oe,Ce),F.setClearColor(Ze,Ke),it!==void 0&&(q.viewport=it),F.toneMapping=et}function Gn(b,W,J){const q=W.isScene===!0?W.overrideMaterial:null;for(let Z=0,Te=b.length;Z<Te;Z++){const De=b[Z],{object:be,geometry:Oe,group:Ce}=De;let et=De.material;et.allowOverride===!0&&q!==null&&(et=q),be.layers.test(J.layers)&&Qt(be,W,J,Oe,et,Ce)}}function Qt(b,W,J,q,Z,Te){b.onBeforeRender(F,W,J,q,Z,Te),b.modelViewMatrix.multiplyMatrices(J.matrixWorldInverse,b.matrixWorld),b.normalMatrix.getNormalMatrix(b.modelViewMatrix),Z.onBeforeRender(F,W,J,q,b,Te),Z.transparent===!0&&Z.side===Pi&&Z.forceSinglePass===!1?(Z.side=Vn,Z.needsUpdate=!0,F.renderBufferDirect(J,W,q,Z,b,Te),Z.side=xr,Z.needsUpdate=!0,F.renderBufferDirect(J,W,q,Z,b,Te),Z.side=Pi):F.renderBufferDirect(J,W,q,Z,b,Te),b.onAfterRender(F,W,J,q,Z,Te)}function Yt(b,W,J){W.isScene!==!0&&(W=_t);const q=$.get(b),Z=y.state.lights,Te=y.state.shadowsArray,De=Z.state.version,be=xe.getParameters(b,Z.state,Te,W,J,y.state.lightProbeGridArray),Oe=xe.getProgramCacheKey(be);let Ce=q.programs;q.environment=b.isMeshStandardMaterial||b.isMeshLambertMaterial||b.isMeshPhongMaterial?W.environment:null,q.fog=W.fog;const et=b.isMeshStandardMaterial||b.isMeshLambertMaterial&&!b.envMap||b.isMeshPhongMaterial&&!b.envMap;q.envMap=ge.get(b.envMap||q.environment,et),q.envMapRotation=q.environment!==null&&b.envMap===null?W.environmentRotation:b.envMapRotation,Ce===void 0&&(b.addEventListener("dispose",vn),Ce=new Map,q.programs=Ce);let it=Ce.get(Oe);if(it!==void 0){if(q.currentProgram===it&&q.lightsStateVersion===De)return Mr(b,be),it}else be.uniforms=xe.getUniforms(b),z!==null&&b.isNodeMaterial&&z.build(b,J,be),b.onBeforeCompile(be,F),it=xe.acquireProgram(be,Oe),Ce.set(Oe,it),q.uniforms=be.uniforms;const He=q.uniforms;return(!b.isShaderMaterial&&!b.isRawShaderMaterial||b.clipping===!0)&&(He.clippingPlanes=ke.uniform),Mr(b,be),q.needsLights=Gs(b),q.lightsStateVersion=De,q.needsLights&&(He.ambientLightColor.value=Z.state.ambient,He.lightProbe.value=Z.state.probe,He.directionalLights.value=Z.state.directional,He.directionalLightShadows.value=Z.state.directionalShadow,He.spotLights.value=Z.state.spot,He.spotLightShadows.value=Z.state.spotShadow,He.rectAreaLights.value=Z.state.rectArea,He.ltc_1.value=Z.state.rectAreaLTC1,He.ltc_2.value=Z.state.rectAreaLTC2,He.pointLights.value=Z.state.point,He.pointLightShadows.value=Z.state.pointShadow,He.hemisphereLights.value=Z.state.hemi,He.directionalShadowMatrix.value=Z.state.directionalShadowMatrix,He.spotLightMatrix.value=Z.state.spotLightMatrix,He.spotLightMap.value=Z.state.spotLightMap,He.pointShadowMatrix.value=Z.state.pointShadowMatrix),q.lightProbeGrid=y.state.lightProbeGridArray.length>0,q.currentProgram=it,q.uniformsList=null,it}function jt(b){if(b.uniformsList===null){const W=b.currentProgram.getUniforms();b.uniformsList=mo.seqWithValue(W.seq,b.uniforms)}return b.uniformsList}function Mr(b,W){const J=$.get(b);J.outputColorSpace=W.outputColorSpace,J.batching=W.batching,J.batchingColor=W.batchingColor,J.instancing=W.instancing,J.instancingColor=W.instancingColor,J.instancingMorph=W.instancingMorph,J.skinning=W.skinning,J.morphTargets=W.morphTargets,J.morphNormals=W.morphNormals,J.morphColors=W.morphColors,J.morphTargetsCount=W.morphTargetsCount,J.numClippingPlanes=W.numClippingPlanes,J.numIntersection=W.numClipIntersection,J.vertexAlphas=W.vertexAlphas,J.vertexTangents=W.vertexTangents,J.toneMapping=W.toneMapping}function Hn(b,W){if(b.length===0)return null;if(b.length===1)return b[0].texture!==null?b[0]:null;S.setFromMatrixPosition(W.matrixWorld);for(let J=0,q=b.length;J<q;J++){const Z=b[J];if(Z.texture!==null&&Z.boundingBox.containsPoint(S))return Z}return null}function yr(b,W,J,q,Z){W.isScene!==!0&&(W=_t),ee.resetTextureUnits();const Te=W.fog,De=q.isMeshStandardMaterial||q.isMeshLambertMaterial||q.isMeshPhongMaterial?W.environment:null,be=j===null?F.outputColorSpace:j.isXRRenderTarget===!0?j.texture.colorSpace:mt.workingColorSpace,Oe=q.isMeshStandardMaterial||q.isMeshLambertMaterial&&!q.envMap||q.isMeshPhongMaterial&&!q.envMap,Ce=ge.get(q.envMap||De,Oe),et=q.vertexColors===!0&&!!J.attributes.color&&J.attributes.color.itemSize===4,it=!!J.attributes.tangent&&(!!q.normalMap||q.anisotropy>0),He=!!J.morphAttributes.position,Mt=!!J.morphAttributes.normal,qt=!!J.morphAttributes.color;let Ht=Ui;q.toneMapped&&(j===null||j.isXRRenderTarget===!0)&&(Ht=F.toneMapping);const Rt=J.morphAttributes.position||J.morphAttributes.normal||J.morphAttributes.color,rn=Rt!==void 0?Rt.length:0,Le=$.get(q),Cn=y.state.lights;if(ot===!0&&(rt===!0||b!==ae)){const Pt=b===ae&&q.id===ne;ke.setState(q,b,Pt)}let dt=!1;q.version===Le.__version?(Le.needsLights&&Le.lightsStateVersion!==Cn.state.version||Le.outputColorSpace!==be||Z.isBatchedMesh&&Le.batching===!1||!Z.isBatchedMesh&&Le.batching===!0||Z.isBatchedMesh&&Le.batchingColor===!0&&Z.colorTexture===null||Z.isBatchedMesh&&Le.batchingColor===!1&&Z.colorTexture!==null||Z.isInstancedMesh&&Le.instancing===!1||!Z.isInstancedMesh&&Le.instancing===!0||Z.isSkinnedMesh&&Le.skinning===!1||!Z.isSkinnedMesh&&Le.skinning===!0||Z.isInstancedMesh&&Le.instancingColor===!0&&Z.instanceColor===null||Z.isInstancedMesh&&Le.instancingColor===!1&&Z.instanceColor!==null||Z.isInstancedMesh&&Le.instancingMorph===!0&&Z.morphTexture===null||Z.isInstancedMesh&&Le.instancingMorph===!1&&Z.morphTexture!==null||Le.envMap!==Ce||q.fog===!0&&Le.fog!==Te||Le.numClippingPlanes!==void 0&&(Le.numClippingPlanes!==ke.numPlanes||Le.numIntersection!==ke.numIntersection)||Le.vertexAlphas!==et||Le.vertexTangents!==it||Le.morphTargets!==He||Le.morphNormals!==Mt||Le.morphColors!==qt||Le.toneMapping!==Ht||Le.morphTargetsCount!==rn||!!Le.lightProbeGrid!=y.state.lightProbeGridArray.length>0)&&(dt=!0):(dt=!0,Le.__version=q.version);let En=Le.currentProgram;dt===!0&&(En=Yt(q,W,Z),z&&q.isNodeMaterial&&z.onUpdateProgram(q,En,Le));let Pn=!1,Wn=!1,er=!1;const yt=En.getUniforms(),Kt=Le.uniforms;if(_.useProgram(En.program)&&(Pn=!0,Wn=!0,er=!0),q.id!==ne&&(ne=q.id,Wn=!0),Le.needsLights){const Pt=Hn(y.state.lightProbeGridArray,Z);Le.lightProbeGrid!==Pt&&(Le.lightProbeGrid=Pt,Wn=!0)}if(Pn||ae!==b){_.buffers.depth.getReversed()&&b.reversedDepth!==!0&&(b._reversedDepth=!0,b.updateProjectionMatrix()),yt.setValue(H,"projectionMatrix",b.projectionMatrix),yt.setValue(H,"viewMatrix",b.matrixWorldInverse);const Dn=yt.map.cameraPosition;Dn!==void 0&&Dn.setValue(H,Gt.setFromMatrixPosition(b.matrixWorld)),w.logarithmicDepthBuffer&&yt.setValue(H,"logDepthBufFC",2/(Math.log(b.far+1)/Math.LN2)),(q.isMeshPhongMaterial||q.isMeshToonMaterial||q.isMeshLambertMaterial||q.isMeshBasicMaterial||q.isMeshStandardMaterial||q.isShaderMaterial)&&yt.setValue(H,"isOrthographic",b.isOrthographicCamera===!0),ae!==b&&(ae=b,Wn=!0,er=!0)}if(Le.needsLights&&(Cn.state.directionalShadowMap.length>0&&yt.setValue(H,"directionalShadowMap",Cn.state.directionalShadowMap,ee),Cn.state.spotShadowMap.length>0&&yt.setValue(H,"spotShadowMap",Cn.state.spotShadowMap,ee),Cn.state.pointShadowMap.length>0&&yt.setValue(H,"pointShadowMap",Cn.state.pointShadowMap,ee)),Z.isSkinnedMesh){yt.setOptional(H,Z,"bindMatrix"),yt.setOptional(H,Z,"bindMatrixInverse");const Pt=Z.skeleton;Pt&&(Pt.boneTexture===null&&Pt.computeBoneTexture(),yt.setValue(H,"boneTexture",Pt.boneTexture,ee))}Z.isBatchedMesh&&(yt.setOptional(H,Z,"batchingTexture"),yt.setValue(H,"batchingTexture",Z._matricesTexture,ee),yt.setOptional(H,Z,"batchingIdTexture"),yt.setValue(H,"batchingIdTexture",Z._indirectTexture,ee),yt.setOptional(H,Z,"batchingColorTexture"),Z._colorsTexture!==null&&yt.setValue(H,"batchingColorTexture",Z._colorsTexture,ee));const Si=J.morphAttributes;if((Si.position!==void 0||Si.normal!==void 0||Si.color!==void 0)&&G.update(Z,J,En),(Wn||Le.receiveShadow!==Z.receiveShadow)&&(Le.receiveShadow=Z.receiveShadow,yt.setValue(H,"receiveShadow",Z.receiveShadow)),(q.isMeshStandardMaterial||q.isMeshLambertMaterial||q.isMeshPhongMaterial)&&q.envMap===null&&W.environment!==null&&(Kt.envMapIntensity.value=W.environmentIntensity),Kt.dfgLUT!==void 0&&(Kt.dfgLUT.value=uy()),Wn){if(yt.setValue(H,"toneMappingExposure",F.toneMappingExposure),Le.needsLights&&Ta(Kt,er),Te&&q.fog===!0&&Ve.refreshFogUniforms(Kt,Te),Ve.refreshMaterialUniforms(Kt,q,he,ue,y.state.transmissionRenderTarget[b.id]),Le.needsLights&&Le.lightProbeGrid){const Pt=Le.lightProbeGrid;Kt.probesSH.value=Pt.texture,Kt.probesMin.value.copy(Pt.boundingBox.min),Kt.probesMax.value.copy(Pt.boundingBox.max),Kt.probesResolution.value.copy(Pt.resolution)}mo.upload(H,jt(Le),Kt,ee)}if(q.isShaderMaterial&&q.uniformsNeedUpdate===!0&&(mo.upload(H,jt(Le),Kt,ee),q.uniformsNeedUpdate=!1),q.isSpriteMaterial&&yt.setValue(H,"center",Z.center),yt.setValue(H,"modelViewMatrix",Z.modelViewMatrix),yt.setValue(H,"normalMatrix",Z.normalMatrix),yt.setValue(H,"modelMatrix",Z.matrixWorld),q.uniformsGroups!==void 0){const Pt=q.uniformsGroups;for(let Dn=0,ci=Pt.length;Dn<ci;Dn++){const Xr=Pt[Dn];de.update(Xr,En),de.bind(Xr,En)}}return En}function Ta(b,W){b.ambientLightColor.needsUpdate=W,b.lightProbe.needsUpdate=W,b.directionalLights.needsUpdate=W,b.directionalLightShadows.needsUpdate=W,b.pointLights.needsUpdate=W,b.pointLightShadows.needsUpdate=W,b.spotLights.needsUpdate=W,b.spotLightShadows.needsUpdate=W,b.rectAreaLights.needsUpdate=W,b.hemisphereLights.needsUpdate=W}function Gs(b){return b.isMeshLambertMaterial||b.isMeshToonMaterial||b.isMeshPhongMaterial||b.isMeshStandardMaterial||b.isShadowMaterial||b.isShaderMaterial&&b.lights===!0}this.getActiveCubeFace=function(){return N},this.getActiveMipmapLevel=function(){return B},this.getRenderTarget=function(){return j},this.setRenderTargetTextures=function(b,W,J){const q=$.get(b);q.__autoAllocateDepthBuffer=b.resolveDepthBuffer===!1,q.__autoAllocateDepthBuffer===!1&&(q.__useRenderToTexture=!1),$.get(b.texture).__webglTexture=W,$.get(b.depthTexture).__webglTexture=q.__autoAllocateDepthBuffer?void 0:J,q.__hasExternalTextures=!0},this.setRenderTargetFramebuffer=function(b,W){const J=$.get(b);J.__webglFramebuffer=W,J.__useDefaultFramebuffer=W===void 0},this.setRenderTarget=function(b,W=0,J=0){j=b,N=W,B=J;let q=null,Z=!1,Te=!1;if(b){const be=$.get(b);if(be.__useDefaultFramebuffer!==void 0){_.bindFramebuffer(H.FRAMEBUFFER,be.__webglFramebuffer),fe.copy(b.viewport),re.copy(b.scissor),le=b.scissorTest,_.viewport(fe),_.scissor(re),_.setScissorTest(le),ne=-1;return}else if(be.__webglFramebuffer===void 0)ee.setupRenderTarget(b);else if(be.__hasExternalTextures)ee.rebindTextures(b,$.get(b.texture).__webglTexture,$.get(b.depthTexture).__webglTexture);else if(b.depthBuffer){const et=b.depthTexture;if(be.__boundDepthTexture!==et){if(et!==null&&$.has(et)&&(b.width!==et.image.width||b.height!==et.image.height))throw new Error("THREE.WebGLRenderer: Attached DepthTexture is initialized to the incorrect size.");ee.setupDepthRenderbuffer(b)}}const Oe=b.texture;(Oe.isData3DTexture||Oe.isDataArrayTexture||Oe.isCompressedArrayTexture)&&(Te=!0);const Ce=$.get(b).__webglFramebuffer;b.isWebGLCubeRenderTarget?(Array.isArray(Ce[W])?q=Ce[W][J]:q=Ce[W],Z=!0):b.samples>0&&ee.useMultisampledRTT(b)===!1?q=$.get(b).__webglMultisampledFramebuffer:Array.isArray(Ce)?q=Ce[J]:q=Ce,fe.copy(b.viewport),re.copy(b.scissor),le=b.scissorTest}else fe.copy(ze).multiplyScalar(he).floor(),re.copy(St).multiplyScalar(he).floor(),le=je;if(J!==0&&(q=O),_.bindFramebuffer(H.FRAMEBUFFER,q)&&_.drawBuffers(b,q),_.viewport(fe),_.scissor(re),_.setScissorTest(le),Z){const be=$.get(b.texture);H.framebufferTexture2D(H.FRAMEBUFFER,H.COLOR_ATTACHMENT0,H.TEXTURE_CUBE_MAP_POSITIVE_X+W,be.__webglTexture,J)}else if(Te){const be=W;for(let Oe=0;Oe<b.textures.length;Oe++){const Ce=$.get(b.textures[Oe]);H.framebufferTextureLayer(H.FRAMEBUFFER,H.COLOR_ATTACHMENT0+Oe,Ce.__webglTexture,J,be)}}else if(b!==null&&J!==0){const be=$.get(b.texture);H.framebufferTexture2D(H.FRAMEBUFFER,H.COLOR_ATTACHMENT0,H.TEXTURE_2D,be.__webglTexture,J)}ne=-1},this.readRenderTargetPixels=function(b,W,J,q,Z,Te,De,be=0){if(!(b&&b.isWebGLRenderTarget)){gt("WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");return}let Oe=$.get(b).__webglFramebuffer;if(b.isWebGLCubeRenderTarget&&De!==void 0&&(Oe=Oe[De]),Oe){_.bindFramebuffer(H.FRAMEBUFFER,Oe);try{const Ce=b.textures[be],et=Ce.format,it=Ce.type;if(b.textures.length>1&&H.readBuffer(H.COLOR_ATTACHMENT0+be),!w.textureFormatReadable(et)){gt("WebGLRenderer.readRenderTargetPixels: renderTarget is not in RGBA or implementation defined format.");return}if(!w.textureTypeReadable(it)){gt("WebGLRenderer.readRenderTargetPixels: renderTarget is not in UnsignedByteType or implementation defined type.");return}W>=0&&W<=b.width-q&&J>=0&&J<=b.height-Z&&H.readPixels(W,J,q,Z,Ee.convert(et),Ee.convert(it),Te)}finally{const Ce=j!==null?$.get(j).__webglFramebuffer:null;_.bindFramebuffer(H.FRAMEBUFFER,Ce)}}},this.readRenderTargetPixelsAsync=async function(b,W,J,q,Z,Te,De,be=0){if(!(b&&b.isWebGLRenderTarget))throw new Error("THREE.WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");let Oe=$.get(b).__webglFramebuffer;if(b.isWebGLCubeRenderTarget&&De!==void 0&&(Oe=Oe[De]),Oe)if(W>=0&&W<=b.width-q&&J>=0&&J<=b.height-Z){_.bindFramebuffer(H.FRAMEBUFFER,Oe);const Ce=b.textures[be],et=Ce.format,it=Ce.type;if(b.textures.length>1&&H.readBuffer(H.COLOR_ATTACHMENT0+be),!w.textureFormatReadable(et))throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in RGBA or implementation defined format.");if(!w.textureTypeReadable(it))throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in UnsignedByteType or implementation defined type.");const He=H.createBuffer();H.bindBuffer(H.PIXEL_PACK_BUFFER,He),H.bufferData(H.PIXEL_PACK_BUFFER,Te.byteLength,H.STREAM_READ),H.readPixels(W,J,q,Z,Ee.convert(et),Ee.convert(it),0);const Mt=j!==null?$.get(j).__webglFramebuffer:null;_.bindFramebuffer(H.FRAMEBUFFER,Mt);const qt=H.fenceSync(H.SYNC_GPU_COMMANDS_COMPLETE,0);return H.flush(),await b0(H,qt,4),H.bindBuffer(H.PIXEL_PACK_BUFFER,He),H.getBufferSubData(H.PIXEL_PACK_BUFFER,0,Te),H.deleteBuffer(He),H.deleteSync(qt),Te}else throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: requested read bounds are out of range.")},this.copyFramebufferToTexture=function(b,W=null,J=0){const q=Math.pow(2,-J),Z=Math.floor(b.image.width*q),Te=Math.floor(b.image.height*q),De=W!==null?W.x:0,be=W!==null?W.y:0;ee.setTexture2D(b,0),H.copyTexSubImage2D(H.TEXTURE_2D,J,0,0,De,be,Z,Te),_.unbindTexture()},this.copyTextureToTexture=function(b,W,J=null,q=null,Z=0,Te=0){let De,be,Oe,Ce,et,it,He,Mt,qt;const Ht=b.isCompressedTexture?b.mipmaps[Te]:b.image;if(J!==null)De=J.max.x-J.min.x,be=J.max.y-J.min.y,Oe=J.isBox3?J.max.z-J.min.z:1,Ce=J.min.x,et=J.min.y,it=J.isBox3?J.min.z:0;else{const Kt=Math.pow(2,-Z);De=Math.floor(Ht.width*Kt),be=Math.floor(Ht.height*Kt),b.isDataArrayTexture?Oe=Ht.depth:b.isData3DTexture?Oe=Math.floor(Ht.depth*Kt):Oe=1,Ce=0,et=0,it=0}q!==null?(He=q.x,Mt=q.y,qt=q.z):(He=0,Mt=0,qt=0);const Rt=Ee.convert(W.format),rn=Ee.convert(W.type);let Le;W.isData3DTexture?(ee.setTexture3D(W,0),Le=H.TEXTURE_3D):W.isDataArrayTexture||W.isCompressedArrayTexture?(ee.setTexture2DArray(W,0),Le=H.TEXTURE_2D_ARRAY):(ee.setTexture2D(W,0),Le=H.TEXTURE_2D),_.activeTexture(H.TEXTURE0),_.pixelStorei(H.UNPACK_FLIP_Y_WEBGL,W.flipY),_.pixelStorei(H.UNPACK_PREMULTIPLY_ALPHA_WEBGL,W.premultiplyAlpha),_.pixelStorei(H.UNPACK_ALIGNMENT,W.unpackAlignment);const Cn=_.getParameter(H.UNPACK_ROW_LENGTH),dt=_.getParameter(H.UNPACK_IMAGE_HEIGHT),En=_.getParameter(H.UNPACK_SKIP_PIXELS),Pn=_.getParameter(H.UNPACK_SKIP_ROWS),Wn=_.getParameter(H.UNPACK_SKIP_IMAGES);_.pixelStorei(H.UNPACK_ROW_LENGTH,Ht.width),_.pixelStorei(H.UNPACK_IMAGE_HEIGHT,Ht.height),_.pixelStorei(H.UNPACK_SKIP_PIXELS,Ce),_.pixelStorei(H.UNPACK_SKIP_ROWS,et),_.pixelStorei(H.UNPACK_SKIP_IMAGES,it);const er=b.isDataArrayTexture||b.isData3DTexture,yt=W.isDataArrayTexture||W.isData3DTexture;if(b.isDepthTexture){const Kt=$.get(b),Si=$.get(W),Pt=$.get(Kt.__renderTarget),Dn=$.get(Si.__renderTarget);_.bindFramebuffer(H.READ_FRAMEBUFFER,Pt.__webglFramebuffer),_.bindFramebuffer(H.DRAW_FRAMEBUFFER,Dn.__webglFramebuffer);for(let ci=0;ci<Oe;ci++)er&&(H.framebufferTextureLayer(H.READ_FRAMEBUFFER,H.COLOR_ATTACHMENT0,$.get(b).__webglTexture,Z,it+ci),H.framebufferTextureLayer(H.DRAW_FRAMEBUFFER,H.COLOR_ATTACHMENT0,$.get(W).__webglTexture,Te,qt+ci)),H.blitFramebuffer(Ce,et,De,be,He,Mt,De,be,H.DEPTH_BUFFER_BIT,H.NEAREST);_.bindFramebuffer(H.READ_FRAMEBUFFER,null),_.bindFramebuffer(H.DRAW_FRAMEBUFFER,null)}else if(Z!==0||b.isRenderTargetTexture||$.has(b)){const Kt=$.get(b),Si=$.get(W);_.bindFramebuffer(H.READ_FRAMEBUFFER,k),_.bindFramebuffer(H.DRAW_FRAMEBUFFER,L);for(let Pt=0;Pt<Oe;Pt++)er?H.framebufferTextureLayer(H.READ_FRAMEBUFFER,H.COLOR_ATTACHMENT0,Kt.__webglTexture,Z,it+Pt):H.framebufferTexture2D(H.READ_FRAMEBUFFER,H.COLOR_ATTACHMENT0,H.TEXTURE_2D,Kt.__webglTexture,Z),yt?H.framebufferTextureLayer(H.DRAW_FRAMEBUFFER,H.COLOR_ATTACHMENT0,Si.__webglTexture,Te,qt+Pt):H.framebufferTexture2D(H.DRAW_FRAMEBUFFER,H.COLOR_ATTACHMENT0,H.TEXTURE_2D,Si.__webglTexture,Te),Z!==0?H.blitFramebuffer(Ce,et,De,be,He,Mt,De,be,H.COLOR_BUFFER_BIT,H.NEAREST):yt?H.copyTexSubImage3D(Le,Te,He,Mt,qt+Pt,Ce,et,De,be):H.copyTexSubImage2D(Le,Te,He,Mt,Ce,et,De,be);_.bindFramebuffer(H.READ_FRAMEBUFFER,null),_.bindFramebuffer(H.DRAW_FRAMEBUFFER,null)}else yt?b.isDataTexture||b.isData3DTexture?H.texSubImage3D(Le,Te,He,Mt,qt,De,be,Oe,Rt,rn,Ht.data):W.isCompressedArrayTexture?H.compressedTexSubImage3D(Le,Te,He,Mt,qt,De,be,Oe,Rt,Ht.data):H.texSubImage3D(Le,Te,He,Mt,qt,De,be,Oe,Rt,rn,Ht):b.isDataTexture?H.texSubImage2D(H.TEXTURE_2D,Te,He,Mt,De,be,Rt,rn,Ht.data):b.isCompressedTexture?H.compressedTexSubImage2D(H.TEXTURE_2D,Te,He,Mt,Ht.width,Ht.height,Rt,Ht.data):H.texSubImage2D(H.TEXTURE_2D,Te,He,Mt,De,be,Rt,rn,Ht);_.pixelStorei(H.UNPACK_ROW_LENGTH,Cn),_.pixelStorei(H.UNPACK_IMAGE_HEIGHT,dt),_.pixelStorei(H.UNPACK_SKIP_PIXELS,En),_.pixelStorei(H.UNPACK_SKIP_ROWS,Pn),_.pixelStorei(H.UNPACK_SKIP_IMAGES,Wn),Te===0&&W.generateMipmaps&&H.generateMipmap(Le),_.unbindTexture()},this.initRenderTarget=function(b){$.get(b).__webglFramebuffer===void 0&&ee.setupRenderTarget(b)},this.initTexture=function(b){b.isCubeTexture?ee.setTextureCube(b,0):b.isData3DTexture?ee.setTexture3D(b,0):b.isDataArrayTexture||b.isCompressedArrayTexture?ee.setTexture2DArray(b,0):ee.setTexture2D(b,0),_.unbindTexture()},this.resetState=function(){N=0,B=0,j=null,_.reset(),Re.reset()},typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}get coordinateSystem(){return Ii}get outputColorSpace(){return this._outputColorSpace}set outputColorSpace(e){this._outputColorSpace=e;const t=this.getContext();t.drawingBufferColorSpace=mt._getDrawingBufferColorSpace(e),t.unpackColorSpace=mt._getUnpackColorSpace()}}const zh={type:"change"},xu={type:"start"},pp={type:"end"},ro=new pu,kh=new hr,hy=Math.cos(70*w0.DEG2RAD),sn=new X,kn=2*Math.PI,It={NONE:-1,ROTATE:0,DOLLY:1,PAN:2,TOUCH_ROTATE:3,TOUCH_PAN:4,TOUCH_DOLLY_PAN:5,TOUCH_DOLLY_ROTATE:6},kl=1e-6;class dy extends m_{constructor(e,t=null){super(e,t),this.state=It.NONE,this.target=new X,this.cursor=new X,this.minDistance=0,this.maxDistance=1/0,this.minZoom=0,this.maxZoom=1/0,this.minTargetRadius=0,this.maxTargetRadius=1/0,this.minPolarAngle=0,this.maxPolarAngle=Math.PI,this.minAzimuthAngle=-1/0,this.maxAzimuthAngle=1/0,this.enableDamping=!1,this.dampingFactor=.05,this.enableZoom=!0,this.zoomSpeed=1,this.enableRotate=!0,this.rotateSpeed=1,this.keyRotateSpeed=1,this.enablePan=!0,this.panSpeed=1,this.screenSpacePanning=!0,this.keyPanSpeed=7,this.zoomToCursor=!1,this.autoRotate=!1,this.autoRotateSpeed=2,this.keys={LEFT:"ArrowLeft",UP:"ArrowUp",RIGHT:"ArrowRight",BOTTOM:"ArrowDown"},this.mouseButtons={LEFT:Es.ROTATE,MIDDLE:Es.DOLLY,RIGHT:Es.PAN},this.touches={ONE:Ms.ROTATE,TWO:Ms.DOLLY_PAN},this.target0=this.target.clone(),this.position0=this.object.position.clone(),this.zoom0=this.object.zoom,this._cursorStyle="auto",this._domElementKeyEvents=null,this._lastPosition=new X,this._lastQuaternion=new vr,this._lastTargetPosition=new X,this._quat=new vr().setFromUnitVectors(e.up,new X(0,1,0)),this._quatInverse=this._quat.clone().invert(),this._spherical=new dh,this._sphericalDelta=new dh,this._scale=1,this._panOffset=new X,this._rotateStart=new $e,this._rotateEnd=new $e,this._rotateDelta=new $e,this._panStart=new $e,this._panEnd=new $e,this._panDelta=new $e,this._dollyStart=new $e,this._dollyEnd=new $e,this._dollyDelta=new $e,this._dollyDirection=new X,this._mouse=new $e,this._performCursorZoom=!1,this._pointers=[],this._pointerPositions={},this._controlActive=!1,this._onPointerMove=my.bind(this),this._onPointerDown=py.bind(this),this._onPointerUp=gy.bind(this),this._onContextMenu=Ey.bind(this),this._onMouseWheel=vy.bind(this),this._onKeyDown=Sy.bind(this),this._onTouchStart=My.bind(this),this._onTouchMove=yy.bind(this),this._onMouseDown=_y.bind(this),this._onMouseMove=xy.bind(this),this._interceptControlDown=by.bind(this),this._interceptControlUp=Ty.bind(this),this.domElement!==null&&this.connect(this.domElement),this.update()}set cursorStyle(e){this._cursorStyle=e,e==="grab"?this.domElement.style.cursor="grab":this.domElement.style.cursor="auto"}get cursorStyle(){return this._cursorStyle}connect(e){super.connect(e),this.domElement.addEventListener("pointerdown",this._onPointerDown),this.domElement.addEventListener("pointercancel",this._onPointerUp),this.domElement.addEventListener("contextmenu",this._onContextMenu),this.domElement.addEventListener("wheel",this._onMouseWheel,{passive:!1}),this.domElement.getRootNode().addEventListener("keydown",this._interceptControlDown,{passive:!0,capture:!0}),this.domElement.style.touchAction="none"}disconnect(){this.domElement.removeEventListener("pointerdown",this._onPointerDown),this.domElement.ownerDocument.removeEventListener("pointermove",this._onPointerMove),this.domElement.ownerDocument.removeEventListener("pointerup",this._onPointerUp),this.domElement.removeEventListener("pointercancel",this._onPointerUp),this.domElement.removeEventListener("wheel",this._onMouseWheel),this.domElement.removeEventListener("contextmenu",this._onContextMenu),this.stopListenToKeyEvents(),this.domElement.getRootNode().removeEventListener("keydown",this._interceptControlDown,{capture:!0}),this.domElement.style.touchAction=""}dispose(){this.disconnect()}getPolarAngle(){return this._spherical.phi}getAzimuthalAngle(){return this._spherical.theta}getDistance(){return this.object.position.distanceTo(this.target)}listenToKeyEvents(e){e.addEventListener("keydown",this._onKeyDown),this._domElementKeyEvents=e}stopListenToKeyEvents(){this._domElementKeyEvents!==null&&(this._domElementKeyEvents.removeEventListener("keydown",this._onKeyDown),this._domElementKeyEvents=null)}saveState(){this.target0.copy(this.target),this.position0.copy(this.object.position),this.zoom0=this.object.zoom}reset(){this.target.copy(this.target0),this.object.position.copy(this.position0),this.object.zoom=this.zoom0,this.object.updateProjectionMatrix(),this.dispatchEvent(zh),this.update(),this.state=It.NONE}pan(e,t){this._pan(e,t),this.update()}dollyIn(e){this._dollyIn(e),this.update()}dollyOut(e){this._dollyOut(e),this.update()}rotateLeft(e){this._rotateLeft(e),this.update()}rotateUp(e){this._rotateUp(e),this.update()}update(e=null){const t=this.object.position;sn.copy(t).sub(this.target),sn.applyQuaternion(this._quat),this._spherical.setFromVector3(sn),this.autoRotate&&this.state===It.NONE&&this._rotateLeft(this._getAutoRotationAngle(e)),this.enableDamping?(this._spherical.theta+=this._sphericalDelta.theta*this.dampingFactor,this._spherical.phi+=this._sphericalDelta.phi*this.dampingFactor):(this._spherical.theta+=this._sphericalDelta.theta,this._spherical.phi+=this._sphericalDelta.phi);let n=this.minAzimuthAngle,r=this.maxAzimuthAngle;isFinite(n)&&isFinite(r)&&(n<-Math.PI?n+=kn:n>Math.PI&&(n-=kn),r<-Math.PI?r+=kn:r>Math.PI&&(r-=kn),n<=r?this._spherical.theta=Math.max(n,Math.min(r,this._spherical.theta)):this._spherical.theta=this._spherical.theta>(n+r)/2?Math.max(n,this._spherical.theta):Math.min(r,this._spherical.theta)),this._spherical.phi=Math.max(this.minPolarAngle,Math.min(this.maxPolarAngle,this._spherical.phi)),this._spherical.makeSafe(),this.enableDamping===!0?this.target.addScaledVector(this._panOffset,this.dampingFactor):this.target.add(this._panOffset),this.target.sub(this.cursor),this.target.clampLength(this.minTargetRadius,this.maxTargetRadius),this.target.add(this.cursor);let s=!1;if(this.zoomToCursor&&this._performCursorZoom||this.object.isOrthographicCamera)this._spherical.radius=this._clampDistance(this._spherical.radius);else{const a=this._spherical.radius;this._spherical.radius=this._clampDistance(this._spherical.radius*this._scale),s=a!=this._spherical.radius}if(sn.setFromSpherical(this._spherical),sn.applyQuaternion(this._quatInverse),t.copy(this.target).add(sn),this.object.lookAt(this.target),this.enableDamping===!0?(this._sphericalDelta.theta*=1-this.dampingFactor,this._sphericalDelta.phi*=1-this.dampingFactor,this._panOffset.multiplyScalar(1-this.dampingFactor)):(this._sphericalDelta.set(0,0,0),this._panOffset.set(0,0,0)),this.zoomToCursor&&this._performCursorZoom){let a=null;if(this.object.isPerspectiveCamera){const o=sn.length();a=this._clampDistance(o*this._scale);const l=o-a;this.object.position.addScaledVector(this._dollyDirection,l),this.object.updateMatrixWorld(),s=!!l}else if(this.object.isOrthographicCamera){const o=new X(this._mouse.x,this._mouse.y,0);o.unproject(this.object);const l=this.object.zoom;this.object.zoom=Math.max(this.minZoom,Math.min(this.maxZoom,this.object.zoom/this._scale)),this.object.updateProjectionMatrix(),s=l!==this.object.zoom;const c=new X(this._mouse.x,this._mouse.y,0);c.unproject(this.object),this.object.position.sub(c).add(o),this.object.updateMatrixWorld(),a=sn.length()}else console.warn("WARNING: OrbitControls.js encountered an unknown camera type - zoom to cursor disabled."),this.zoomToCursor=!1;a!==null&&(this.screenSpacePanning?this.target.set(0,0,-1).transformDirection(this.object.matrix).multiplyScalar(a).add(this.object.position):(ro.origin.copy(this.object.position),ro.direction.set(0,0,-1).transformDirection(this.object.matrix),Math.abs(this.object.up.dot(ro.direction))<hy?this.object.lookAt(this.target):(kh.setFromNormalAndCoplanarPoint(this.object.up,this.target),ro.intersectPlane(kh,this.target))))}else if(this.object.isOrthographicCamera){const a=this.object.zoom;this.object.zoom=Math.max(this.minZoom,Math.min(this.maxZoom,this.object.zoom/this._scale)),a!==this.object.zoom&&(this.object.updateProjectionMatrix(),s=!0)}return this._scale=1,this._performCursorZoom=!1,s||this._lastPosition.distanceToSquared(this.object.position)>kl||8*(1-this._lastQuaternion.dot(this.object.quaternion))>kl||this._lastTargetPosition.distanceToSquared(this.target)>kl?(this.dispatchEvent(zh),this._lastPosition.copy(this.object.position),this._lastQuaternion.copy(this.object.quaternion),this._lastTargetPosition.copy(this.target),!0):!1}_getAutoRotationAngle(e){return e!==null?kn/60*this.autoRotateSpeed*e:kn/60/60*this.autoRotateSpeed}_getZoomScale(e){const t=Math.abs(e*.01);return Math.pow(.95,this.zoomSpeed*t)}_rotateLeft(e){this._sphericalDelta.theta-=e}_rotateUp(e){this._sphericalDelta.phi-=e}_panLeft(e,t){sn.setFromMatrixColumn(t,0),sn.multiplyScalar(-e),this._panOffset.add(sn)}_panUp(e,t){this.screenSpacePanning===!0?sn.setFromMatrixColumn(t,1):(sn.setFromMatrixColumn(t,0),sn.crossVectors(this.object.up,sn)),sn.multiplyScalar(e),this._panOffset.add(sn)}_pan(e,t){const n=this.domElement;if(this.object.isPerspectiveCamera){const r=this.object.position;sn.copy(r).sub(this.target);let s=sn.length();s*=Math.tan(this.object.fov/2*Math.PI/180),this._panLeft(2*e*s/n.clientHeight,this.object.matrix),this._panUp(2*t*s/n.clientHeight,this.object.matrix)}else this.object.isOrthographicCamera?(this._panLeft(e*(this.object.right-this.object.left)/this.object.zoom/n.clientWidth,this.object.matrix),this._panUp(t*(this.object.top-this.object.bottom)/this.object.zoom/n.clientHeight,this.object.matrix)):(console.warn("WARNING: OrbitControls.js encountered an unknown camera type - pan disabled."),this.enablePan=!1)}_dollyOut(e){this.object.isPerspectiveCamera||this.object.isOrthographicCamera?this._scale/=e:(console.warn("WARNING: OrbitControls.js encountered an unknown camera type - dolly/zoom disabled."),this.enableZoom=!1)}_dollyIn(e){this.object.isPerspectiveCamera||this.object.isOrthographicCamera?this._scale*=e:(console.warn("WARNING: OrbitControls.js encountered an unknown camera type - dolly/zoom disabled."),this.enableZoom=!1)}_updateZoomParameters(e,t){if(!this.zoomToCursor)return;this._performCursorZoom=!0;const n=this.domElement.getBoundingClientRect(),r=e-n.left,s=t-n.top,a=n.width,o=n.height;this._mouse.x=r/a*2-1,this._mouse.y=-(s/o)*2+1,this._dollyDirection.set(this._mouse.x,this._mouse.y,1).unproject(this.object).sub(this.object.position).normalize()}_clampDistance(e){return Math.max(this.minDistance,Math.min(this.maxDistance,e))}_handleMouseDownRotate(e){this._rotateStart.set(e.clientX,e.clientY)}_handleMouseDownDolly(e){this._updateZoomParameters(e.clientX,e.clientX),this._dollyStart.set(e.clientX,e.clientY)}_handleMouseDownPan(e){this._panStart.set(e.clientX,e.clientY)}_handleMouseMoveRotate(e){this._rotateEnd.set(e.clientX,e.clientY),this._rotateDelta.subVectors(this._rotateEnd,this._rotateStart).multiplyScalar(this.rotateSpeed);const t=this.domElement;this._rotateLeft(kn*this._rotateDelta.x/t.clientHeight),this._rotateUp(kn*this._rotateDelta.y/t.clientHeight),this._rotateStart.copy(this._rotateEnd),this.update()}_handleMouseMoveDolly(e){this._dollyEnd.set(e.clientX,e.clientY),this._dollyDelta.subVectors(this._dollyEnd,this._dollyStart),this._dollyDelta.y>0?this._dollyOut(this._getZoomScale(this._dollyDelta.y)):this._dollyDelta.y<0&&this._dollyIn(this._getZoomScale(this._dollyDelta.y)),this._dollyStart.copy(this._dollyEnd),this.update()}_handleMouseMovePan(e){this._panEnd.set(e.clientX,e.clientY),this._panDelta.subVectors(this._panEnd,this._panStart).multiplyScalar(this.panSpeed),this._pan(this._panDelta.x,this._panDelta.y),this._panStart.copy(this._panEnd),this.update()}_handleMouseWheel(e){this._updateZoomParameters(e.clientX,e.clientY),e.deltaY<0?this._dollyIn(this._getZoomScale(e.deltaY)):e.deltaY>0&&this._dollyOut(this._getZoomScale(e.deltaY)),this.update()}_handleKeyDown(e){let t=!1;switch(e.code){case this.keys.UP:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateUp(kn*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(0,this.keyPanSpeed),t=!0;break;case this.keys.BOTTOM:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateUp(-kn*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(0,-this.keyPanSpeed),t=!0;break;case this.keys.LEFT:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateLeft(kn*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(this.keyPanSpeed,0),t=!0;break;case this.keys.RIGHT:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateLeft(-kn*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(-this.keyPanSpeed,0),t=!0;break}t&&(e.preventDefault(),this.update())}_handleTouchStartRotate(e){if(this._pointers.length===1)this._rotateStart.set(e.pageX,e.pageY);else{const t=this._getSecondPointerPosition(e),n=.5*(e.pageX+t.x),r=.5*(e.pageY+t.y);this._rotateStart.set(n,r)}}_handleTouchStartPan(e){if(this._pointers.length===1)this._panStart.set(e.pageX,e.pageY);else{const t=this._getSecondPointerPosition(e),n=.5*(e.pageX+t.x),r=.5*(e.pageY+t.y);this._panStart.set(n,r)}}_handleTouchStartDolly(e){const t=this._getSecondPointerPosition(e),n=e.pageX-t.x,r=e.pageY-t.y,s=Math.sqrt(n*n+r*r);this._dollyStart.set(0,s)}_handleTouchStartDollyPan(e){this.enableZoom&&this._handleTouchStartDolly(e),this.enablePan&&this._handleTouchStartPan(e)}_handleTouchStartDollyRotate(e){this.enableZoom&&this._handleTouchStartDolly(e),this.enableRotate&&this._handleTouchStartRotate(e)}_handleTouchMoveRotate(e){if(this._pointers.length==1)this._rotateEnd.set(e.pageX,e.pageY);else{const n=this._getSecondPointerPosition(e),r=.5*(e.pageX+n.x),s=.5*(e.pageY+n.y);this._rotateEnd.set(r,s)}this._rotateDelta.subVectors(this._rotateEnd,this._rotateStart).multiplyScalar(this.rotateSpeed);const t=this.domElement;this._rotateLeft(kn*this._rotateDelta.x/t.clientHeight),this._rotateUp(kn*this._rotateDelta.y/t.clientHeight),this._rotateStart.copy(this._rotateEnd)}_handleTouchMovePan(e){if(this._pointers.length===1)this._panEnd.set(e.pageX,e.pageY);else{const t=this._getSecondPointerPosition(e),n=.5*(e.pageX+t.x),r=.5*(e.pageY+t.y);this._panEnd.set(n,r)}this._panDelta.subVectors(this._panEnd,this._panStart).multiplyScalar(this.panSpeed),this._pan(this._panDelta.x,this._panDelta.y),this._panStart.copy(this._panEnd)}_handleTouchMoveDolly(e){const t=this._getSecondPointerPosition(e),n=e.pageX-t.x,r=e.pageY-t.y,s=Math.sqrt(n*n+r*r);this._dollyEnd.set(0,s),this._dollyDelta.set(0,Math.pow(this._dollyEnd.y/this._dollyStart.y,this.zoomSpeed)),this._dollyOut(this._dollyDelta.y),this._dollyStart.copy(this._dollyEnd);const a=(e.pageX+t.x)*.5,o=(e.pageY+t.y)*.5;this._updateZoomParameters(a,o)}_handleTouchMoveDollyPan(e){this.enableZoom&&this._handleTouchMoveDolly(e),this.enablePan&&this._handleTouchMovePan(e)}_handleTouchMoveDollyRotate(e){this.enableZoom&&this._handleTouchMoveDolly(e),this.enableRotate&&this._handleTouchMoveRotate(e)}_addPointer(e){this._pointers.push(e.pointerId)}_removePointer(e){delete this._pointerPositions[e.pointerId];for(let t=0;t<this._pointers.length;t++)if(this._pointers[t]==e.pointerId){this._pointers.splice(t,1);return}}_isTrackingPointer(e){for(let t=0;t<this._pointers.length;t++)if(this._pointers[t]==e.pointerId)return!0;return!1}_trackPointer(e){let t=this._pointerPositions[e.pointerId];t===void 0&&(t=new $e,this._pointerPositions[e.pointerId]=t),t.set(e.pageX,e.pageY)}_getSecondPointerPosition(e){const t=e.pointerId===this._pointers[0]?this._pointers[1]:this._pointers[0];return this._pointerPositions[t]}_customWheelEvent(e){const t=e.deltaMode,n={clientX:e.clientX,clientY:e.clientY,deltaY:e.deltaY};switch(t){case 1:n.deltaY*=16;break;case 2:n.deltaY*=100;break}return e.ctrlKey&&!this._controlActive&&(n.deltaY*=10),n}}function py(i){this.enabled!==!1&&(this._pointers.length===0&&(this.domElement.setPointerCapture(i.pointerId),this.domElement.ownerDocument.addEventListener("pointermove",this._onPointerMove),this.domElement.ownerDocument.addEventListener("pointerup",this._onPointerUp)),!this._isTrackingPointer(i)&&(this._addPointer(i),i.pointerType==="touch"?this._onTouchStart(i):this._onMouseDown(i),this._cursorStyle==="grab"&&(this.domElement.style.cursor="grabbing")))}function my(i){this.enabled!==!1&&(i.pointerType==="touch"?this._onTouchMove(i):this._onMouseMove(i))}function gy(i){switch(this._removePointer(i),this._pointers.length){case 0:this.domElement.releasePointerCapture(i.pointerId),this.domElement.ownerDocument.removeEventListener("pointermove",this._onPointerMove),this.domElement.ownerDocument.removeEventListener("pointerup",this._onPointerUp),this.dispatchEvent(pp),this.state=It.NONE,this._cursorStyle==="grab"&&(this.domElement.style.cursor="grab");break;case 1:const e=this._pointers[0],t=this._pointerPositions[e];this._onTouchStart({pointerId:e,pageX:t.x,pageY:t.y});break}}function _y(i){let e;switch(i.button){case 0:e=this.mouseButtons.LEFT;break;case 1:e=this.mouseButtons.MIDDLE;break;case 2:e=this.mouseButtons.RIGHT;break;default:e=-1}switch(e){case Es.DOLLY:if(this.enableZoom===!1)return;this._handleMouseDownDolly(i),this.state=It.DOLLY;break;case Es.ROTATE:if(i.ctrlKey||i.metaKey||i.shiftKey){if(this.enablePan===!1)return;this._handleMouseDownPan(i),this.state=It.PAN}else{if(this.enableRotate===!1)return;this._handleMouseDownRotate(i),this.state=It.ROTATE}break;case Es.PAN:if(i.ctrlKey||i.metaKey||i.shiftKey){if(this.enableRotate===!1)return;this._handleMouseDownRotate(i),this.state=It.ROTATE}else{if(this.enablePan===!1)return;this._handleMouseDownPan(i),this.state=It.PAN}break;default:this.state=It.NONE}this.state!==It.NONE&&this.dispatchEvent(xu)}function xy(i){switch(this.state){case It.ROTATE:if(this.enableRotate===!1)return;this._handleMouseMoveRotate(i);break;case It.DOLLY:if(this.enableZoom===!1)return;this._handleMouseMoveDolly(i);break;case It.PAN:if(this.enablePan===!1)return;this._handleMouseMovePan(i);break}}function vy(i){this.enabled===!1||this.enableZoom===!1||this.state!==It.NONE||(i.preventDefault(),this.dispatchEvent(xu),this._handleMouseWheel(this._customWheelEvent(i)),this.dispatchEvent(pp))}function Sy(i){this.enabled!==!1&&this._handleKeyDown(i)}function My(i){switch(this._trackPointer(i),this._pointers.length){case 1:switch(this.touches.ONE){case Ms.ROTATE:if(this.enableRotate===!1)return;this._handleTouchStartRotate(i),this.state=It.TOUCH_ROTATE;break;case Ms.PAN:if(this.enablePan===!1)return;this._handleTouchStartPan(i),this.state=It.TOUCH_PAN;break;default:this.state=It.NONE}break;case 2:switch(this.touches.TWO){case Ms.DOLLY_PAN:if(this.enableZoom===!1&&this.enablePan===!1)return;this._handleTouchStartDollyPan(i),this.state=It.TOUCH_DOLLY_PAN;break;case Ms.DOLLY_ROTATE:if(this.enableZoom===!1&&this.enableRotate===!1)return;this._handleTouchStartDollyRotate(i),this.state=It.TOUCH_DOLLY_ROTATE;break;default:this.state=It.NONE}break;default:this.state=It.NONE}this.state!==It.NONE&&this.dispatchEvent(xu)}function yy(i){switch(this._trackPointer(i),this.state){case It.TOUCH_ROTATE:if(this.enableRotate===!1)return;this._handleTouchMoveRotate(i),this.update();break;case It.TOUCH_PAN:if(this.enablePan===!1)return;this._handleTouchMovePan(i),this.update();break;case It.TOUCH_DOLLY_PAN:if(this.enableZoom===!1&&this.enablePan===!1)return;this._handleTouchMoveDollyPan(i),this.update();break;case It.TOUCH_DOLLY_ROTATE:if(this.enableZoom===!1&&this.enableRotate===!1)return;this._handleTouchMoveDollyRotate(i),this.update();break;default:this.state=It.NONE}}function Ey(i){this.enabled!==!1&&i.preventDefault()}function by(i){i.key==="Control"&&(this._controlActive=!0,this.domElement.getRootNode().addEventListener("keyup",this._interceptControlUp,{passive:!0,capture:!0}))}function Ty(i){i.key==="Control"&&(this._controlActive=!1,this.domElement.getRootNode().removeEventListener("keyup",this._interceptControlUp,{passive:!0,capture:!0}))}const Ay=[{topic:"/camera/front/image_raw/compressed",label:"front",position:[0,.3,1.5],lookAt:[0,-.1,3.5],fovDeg:90,color:"#00b4d8"},{topic:"/camera/side/image_raw/compressed",label:"side",position:[0,1,1.2],lookAt:[3,.5,1],fovDeg:90,color:"#ff6b6b"},{topic:"/camera/down/image_raw/compressed",label:"down",position:[0,.5,1.6],lookAt:[0,-2,.5],fovDeg:90,color:"#51cf66"}];function wy(i,e,t,n,r=.2,s=2){const a=new ha,o=new X(...i),c=new X(...e).clone().sub(o).normalize(),f=o.clone().add(c.clone().multiplyScalar(r)),h=o.clone().add(c.clone().multiplyScalar(s)),u=new X(0,1,0);Math.abs(c.dot(u))>.99&&u.set(1,0,0);const d=c.clone().cross(u).normalize(),x=d.clone().cross(c).normalize(),E=t*Math.PI/180,m=Math.tan(E/2)*r,p=Math.tan(E/2)*s,T=m,P=p,S=f.clone().add(x.clone().multiplyScalar(m)).add(d.clone().multiplyScalar(-T)),C=f.clone().add(x.clone().multiplyScalar(m)).add(d.clone().multiplyScalar(T)),y=f.clone().add(x.clone().multiplyScalar(-m)).add(d.clone().multiplyScalar(-T)),I=f.clone().add(x.clone().multiplyScalar(-m)).add(d.clone().multiplyScalar(T)),v=h.clone().add(x.clone().multiplyScalar(p)).add(d.clone().multiplyScalar(-P)),A=h.clone().add(x.clone().multiplyScalar(p)).add(d.clone().multiplyScalar(P)),F=h.clone().add(x.clone().multiplyScalar(-p)).add(d.clone().multiplyScalar(-P)),D=h.clone().add(x.clone().multiplyScalar(-p)).add(d.clone().multiplyScalar(P)),z=new zo({color:n,linewidth:1,transparent:!0,opacity:.7});a.add(new po(new gn().setFromPoints([S,C,I,y,S]),z)),a.add(new po(new gn().setFromPoints([v,A,D,F,v]),z)),[S,C,y,I].forEach((j,ne)=>{const ae=[v,A,F,D][ne];a.add(new po(new gn().setFromPoints([j,ae]),z))});const O=new mu({color:n,transparent:!0,opacity:.08,side:Pi,depthWrite:!1}),k=new gn,L=new Float32Array([S.x,S.y,S.z,C.x,C.y,C.z,y.x,y.y,y.z,C.x,C.y,C.z,I.x,I.y,I.z,y.x,y.y,y.z]);k.setAttribute("position",new Qn(L,3)),a.add(new vi(k,O));const N=new gn,B=new Float32Array([v.x,v.y,v.z,A.x,A.y,A.z,F.x,F.y,F.z,A.x,A.y,A.z,D.x,D.y,D.z,F.x,F.y,F.z]);return N.setAttribute("position",new Qn(B,3)),a.add(new vi(N,O)),a}function Ry(i,e,t){const n=document.createElement("canvas");n.width=128,n.height=32;const r=n.getContext("2d");r.fillStyle=t,r.font="bold 16px monospace",r.textAlign="center",r.fillText(i,64,22);const s=new Q0(n);s.minFilter=yn;const a=new jd({map:s,depthTest:!1,depthWrite:!1}),o=new q0(a);return o.position.copy(e),o.scale.set(1.2,.3,1),o}function Cy({cameras:i=Ay}){const e=Ae.useRef(null);return Ae.useEffect(()=>{if(!e.current)return;const t=e.current,n=t.clientWidth||640,r=t.clientHeight||480,s=new V0;s.background=new ft(1710638);const a=new oi(50,n/r,.1,100);a.position.set(4,5,6),a.lookAt(0,.3,1.2);const o=new fy({antialias:!0});o.setSize(n,r),o.setPixelRatio(Math.min(window.devicePixelRatio,2)),t.appendChild(o.domElement);const l=new dy(a,o.domElement);l.enableDamping=!0,l.dampingFactor=.1,l.target.set(0,.3,1.2),s.add(new u_(4210784));const c=new hh(16777215,1.2);c.position.set(5,10,7),s.add(c);const f=new hh(8947967,.3);f.position.set(-3,-1,-5),s.add(f),s.add(new d_(10,20,4473992,3355494)),s.add(new p_(3)),i.forEach(E=>{const m=wy(E.position,E.lookAt,E.fovDeg,E.color);s.add(m);const p=new X(...E.position).add(new X(0,.4,0));s.add(Ry(E.label,p,E.color))});let h=!0,u=!0;l.addEventListener("change",()=>{u=!0});const d=()=>{if(!h)return;(l.update()||u)&&(o.render(s,a),u=!1),requestAnimationFrame(d)};d();const x=()=>{const E=t.clientWidth,m=t.clientHeight;E===0||m===0||(a.aspect=E/m,a.updateProjectionMatrix(),o.setSize(E,m),u=!0)};return window.addEventListener("resize",x),()=>{h=!1,window.removeEventListener("resize",x),o.dispose(),t.contains(o.domElement)&&t.removeChild(o.domElement)}},[i]),se.jsx("div",{ref:e,style:{width:"100%",height:"100%",minHeight:300,borderRadius:"var(--radius-md)",overflow:"hidden"}})}const{Text:Vh}=Gh;function Py(i){if(!i)return"-";const e=i/1e9;if(e>1)return`${e.toFixed(1)} GB`;const t=i/1e6;return t>1?`${t.toFixed(0)} MB`:`${(i/1e3).toFixed(0)} KB`}function Dy(i){if(!i)return"-";const e=Math.round(i/1e3),t=Math.floor(e/60),n=e%60;return t>0?`${t}m ${n}s`:`${n}s`}function Ly({channels:i,activeTopics:e,onToggle:t,metadata:n}){const[r,s]=Ae.useState("3d"),a=Ae.useMemo(()=>{const o=Lf(500,.3),l=Lf(500,.8);return[o[0],o[1],l[1]]},[]);return se.jsxs("div",{style:{padding:16,display:"flex",flexDirection:"column",gap:16,overflow:"auto"},children:[se.jsxs("div",{children:[se.jsx(Vh,{strong:!0,style:{fontSize:"var(--font-size-base)",color:"var(--gray-800)"},children:"MCAP 文件信息"}),se.jsxs(Ks,{column:1,size:"small",style:{marginTop:8},styles:{label:{fontSize:"var(--font-size-sm)",color:"var(--gray-500)",paddingBottom:4},content:{fontSize:"var(--font-size-sm)",fontFamily:"var(--font-mono)",color:"var(--gray-700)"}},children:[se.jsxs(Ks.Item,{label:"资产 ID",children:[n.assetId.slice(0,16),"…"]}),se.jsx(Ks.Item,{label:"时长",children:Dy(n.durationMs)}),se.jsx(Ks.Item,{label:"大小",children:Py(n.fileSize)}),se.jsx(Ks.Item,{label:"通道数",children:n.channelCount})]})]}),se.jsx(Hu,{style:{margin:0}}),se.jsxs("div",{children:[se.jsx(Vh,{strong:!0,style:{fontSize:"var(--font-size-base)",color:"var(--gray-800)"},children:"通道列表"}),se.jsx("div",{style:{marginTop:8,display:"flex",flexDirection:"column",gap:4},children:i.map(o=>se.jsxs("button",{type:"button",style:{display:"flex",alignItems:"center",gap:8,padding:"6px 8px",borderRadius:"var(--radius-sm)",background:e.includes(o.topic)?"var(--color-primary-light)":"transparent",cursor:"pointer",transition:"background 150ms ease",border:"none",width:"100%",textAlign:"left"},onClick:()=>t(o.topic),children:[se.jsx(Ip,{checked:e.includes(o.topic)}),se.jsxs("span",{style:{fontSize:"var(--font-size-sm)",color:"var(--gray-700)"},children:["📷 ",o.label]}),se.jsx("span",{style:{marginLeft:"auto",fontSize:"var(--font-size-xs)",color:"var(--gray-400)",fontFamily:"var(--font-mono)"},children:o.topic.split("/").filter(Boolean).pop()})]},o.topic))})]}),se.jsx(Hu,{style:{margin:0}}),se.jsxs("div",{children:[se.jsx(Up,{value:r,onChange:o=>s(o),options:[{value:"3d",label:"3D"},{value:"chart",label:"图表"}],size:"small",block:!0,style:{marginBottom:8}}),r==="3d"?se.jsx("div",{style:{height:300,borderRadius:"var(--radius-md)",overflow:"hidden"},children:se.jsx(Cy,{})}):r==="chart"?se.jsx(Hg,{title:"IMU 加速度",data:a,series:["X轴","Y轴"],height:280}):null]})]})}const Iy=Ae.memo(Ly);function Uy(i){const e=Math.floor(i/3600),t=Math.floor(i%3600/60),n=Math.floor(i%60);return e>0?`${e}:${t.toString().padStart(2,"0")}:${n.toString().padStart(2,"0")}`:`${t.toString().padStart(2,"0")}:${n.toString().padStart(2,"0")}`}function Ny({currentTime:i,durationSec:e,range:t,onSeek:n,onRangeChange:r}){const s=Ae.useRef(null),[a,o]=Ae.useState(null),[l,c]=Ae.useState(null),[f,h]=Ae.useState(0),u=Ae.useRef({target:null,startX:0,startSec:0}),d=Ae.useRef(n),x=Ae.useRef(r);d.current=n,x.current=r;const E=Ae.useCallback(y=>{if(!s.current)return 0;const I=s.current.getBoundingClientRect();return Math.max(0,Math.min(e,(y-I.left)/I.width*e))},[e]),m=Ae.useCallback((y,I)=>{y.preventDefault();const v=E(y.clientX);u.current={target:I,startX:y.clientX,startSec:v,startRange:t?{...t}:void 0},o(I);const A=D=>{var k,L;const z=u.current,O=E(D.clientX);switch(z.target){case"playhead":d.current(O);break;case"rangeStart":{if(!z.startRange)break;const N=z.startRange.endSec,B=Math.min(O,N-.1);(k=x.current)==null||k.call(x,{startSec:B,endSec:N});break}case"rangeEnd":{if(!z.startRange)break;const N=z.startRange.startSec,B=Math.max(O,N+.1);(L=x.current)==null||L.call(x,{startSec:N,endSec:B});break}}},F=()=>{u.current={target:null,startX:0,startSec:0},o(null),window.removeEventListener("mousemove",A),window.removeEventListener("mouseup",F)};window.addEventListener("mousemove",A),window.addEventListener("mouseup",F)},[E,t]),p=e>0?i/e*100:0,T=t&&e>0,P=T?t.startSec/e*100:0,S=T?t.endSec/e*100:0,C=20;return se.jsxs("div",{role:"slider","aria-valuenow":currentSec,"aria-valuemin":0,"aria-valuemax":e,tabIndex:0,ref:s,style:{flex:1,height:32,position:"relative",cursor:"pointer",userSelect:"none",margin:"0 4px",touchAction:"none"},onMouseDown:y=>{u.current.target||d.current(E(y.clientX))},onMouseMove:y=>{var I;a||(c(E(y.clientX)),h(y.clientX-(((I=s.current)==null?void 0:I.getBoundingClientRect().left)||0)))},onMouseLeave:()=>c(null),children:[se.jsx("div",{style:{position:"absolute",left:0,right:0,top:"50%",height:4,marginTop:-2,background:a?"var(--gray-300)":"var(--gray-200)",borderRadius:2,transition:a?"none":"background 0.15s"}}),l!==null&&!a&&se.jsx("div",{style:{position:"absolute",bottom:28,left:`calc(${f}px - 30px)`,background:"rgba(0,0,0,0.75)",color:"#fff",padding:"2px 6px",borderRadius:4,fontSize:10,fontFamily:"var(--font-mono)",whiteSpace:"nowrap",pointerEvents:"none",zIndex:10},children:Uy(l)}),T&&se.jsx("div",{style:{position:"absolute",left:`${P}%`,width:`${S-P}%`,top:"50%",height:4,marginTop:-2,background:"rgba(255, 68, 68, 0.35)",borderRadius:2,pointerEvents:"none",transition:a?"none":"left 0.1s, width 0.1s"}}),T&&se.jsx("div",{role:"slider","aria-valuenow":t.startSec,"aria-valuemin":0,"aria-valuemax":e,tabIndex:0,onMouseDown:y=>m(y,"rangeStart"),style:{position:"absolute",left:`${P}%`,top:"50%",width:C,height:C,marginLeft:-C/2,marginTop:-C/2,border:"2px solid #00bcd4",background:a==="rangeStart"?"#00bcd4":"#fff",borderRadius:"50%",cursor:"ew-resize",zIndex:2,boxSizing:"border-box",transition:a==="rangeStart"?"none":"transform 0.1s, background 0.1s"}}),T&&se.jsx("div",{role:"slider","aria-valuenow":t.endSec,"aria-valuemin":0,"aria-valuemax":e,tabIndex:0,onMouseDown:y=>m(y,"rangeEnd"),style:{position:"absolute",left:`${S}%`,top:"50%",width:C,height:C,marginLeft:-C/2,marginTop:-C/2,border:"2px solid var(--color-accent)",background:a==="rangeEnd"?"var(--color-accent)":"#fff",borderRadius:"50%",cursor:"ew-resize",zIndex:2,boxSizing:"border-box",transition:a==="rangeEnd"?"none":"transform 0.1s, background 0.1s"}}),se.jsx("div",{role:"slider","aria-valuenow":currentSec,"aria-valuemin":0,"aria-valuemax":e,tabIndex:0,onMouseDown:y=>m(y,"playhead"),style:{position:"absolute",left:`${p}%`,top:"50%",width:C+2,height:C+2,marginLeft:-22/2,marginTop:-22/2,border:"3px solid #fff",background:a==="playhead"?"#cc0000":"#ff4444",borderRadius:"50%",cursor:"grab",zIndex:3,boxSizing:"border-box",boxShadow:a==="playhead"?"0 0 6px rgba(255,68,68,0.6)":"0 1px 3px rgba(0,0,0,0.3)",transition:a==="playhead"?"none":"box-shadow 0.15s"}})]})}function Vl(i){return`${Math.floor(i/60).toString().padStart(2,"0")}:${Math.floor(i%60).toString().padStart(2,"0")}`}function Fy(i){return new Date(i/1e6).toISOString().slice(11,23)}const Oy=[.25,.5,1,2,4,8];function By({currentTime:i,durationSec:e,onSeek:t}){const[n,r]=Ae.useState(!1),[s,a]=Ae.useState(""),o=()=>{r(!1);const l=s.trim();if(!l)return;let c=0;const f=l.match(/^(\d+):(\d+)$/);f?c=parseInt(f[1],10)*60+parseInt(f[2],10):c=parseFloat(l),Number.isFinite(c)&&c>=0&&t(Math.min(c,e))};return n?se.jsx(so,{size:"small",value:s,onChange:l=>a(l.target.value),onPressEnter:o,onBlur:o,autoFocus:!0,style:{width:100,fontFamily:"var(--font-mono)",fontSize:"var(--font-size-sm)"},placeholder:"MM:SS"}):se.jsxs("button",{type:"button",onClick:()=>{a(Vl(i)),r(!0)},style:{cursor:"pointer",fontSize:"var(--font-size-sm)",fontFamily:"var(--font-mono)",color:"var(--gray-700)",border:"none",background:"transparent",borderBottom:"1px dashed transparent",padding:0},title:"点击输入时间",children:[Vl(i)," / ",Vl(e)]})}function zy({currentTime:i,durationSec:e,playing:t,playbackRate:n,coverMode:r,onToggleCover:s,startTimestampNs:a,range:o,onRangeChange:l,onPlayPause:c,onSeek:f,onChangeRate:h,onFullscreen:u}){const d=a?a+i*1e9:0;return se.jsx("div",{style:{borderTop:"1px solid var(--gray-200)",padding:"var(--space-3) var(--space-6)",background:"#fff",display:"flex",flexDirection:"column",gap:8},children:se.jsxs("div",{style:{display:"flex",alignItems:"center",gap:8,width:"100%"},children:[se.jsx(pr,{title:"前一帧",children:se.jsx(Ci,{type:"text",icon:se.jsx(Np,{}),size:"small",onClick:()=>f(Math.max(0,i-1/30))})}),se.jsx(Ci,{type:"primary",shape:"circle",icon:t?se.jsx(Fp,{}):se.jsx(Op,{}),onClick:c,size:"small"}),se.jsx(pr,{title:"后一帧",children:se.jsx(Ci,{type:"text",icon:se.jsx(Bp,{}),size:"small",onClick:()=>f(Math.min(e,i+1/30))})}),se.jsx(zp,{value:n,onChange:h,size:"small",style:{width:58},options:Oy.map(x=>({value:x,label:`${x}x`}))}),se.jsx("div",{style:{flex:1,minWidth:0},children:se.jsx(Ny,{currentTime:i,durationSec:e,range:o?{startSec:o.startSec,endSec:o.endSec}:null,onSeek:f,onRangeChange:l})}),se.jsx(By,{currentTime:i,durationSec:e,onSeek:f}),d>0&&se.jsx("span",{style:{fontSize:"var(--font-size-xs)",fontFamily:"var(--font-mono)",color:"var(--gray-400)",whiteSpace:"nowrap",minWidth:90},children:Fy(d)}),se.jsx(pr,{title:r?"完整画面":"裁剪满屏",children:se.jsx(Ci,{type:"text",size:"small",icon:r?se.jsx(kp,{style:{color:"var(--gray-500)"}}):se.jsx(Vp,{style:{color:"var(--color-primary)"}}),onClick:s})}),u&&se.jsx(pr,{title:"全屏",children:se.jsx(Ci,{type:"text",icon:se.jsx(Hh,{}),size:"small",onClick:u})})]})})}const ky=[];function Vy({src:i,currentTime:e,playing:t=!1,objectFit:n="cover",onTimeUpdate:r,boxes:s=ky,gazePoint:a}){const o=Ae.useRef(null),l=Ae.useRef(null),c=Ae.useRef(null),f=Ae.useRef({w:0,h:0}),h=Ae.useRef(0),[u,d]=Ae.useState(i?"loading":"idle"),[x,E]=Ae.useState("");Ae.useEffect(()=>{d(i?"loading":"idle"),E(""),h.current=0},[i]);const m=Ae.useCallback(L=>{switch(L){case"canplay":d("playing");break;case"playing":{d("playing");const N=l.current;N&&e>0&&Math.abs(N.currentTime-e)>.5&&(N.currentTime=e);break}case"waiting":d("loading");break;case"stalled":d("loading");break}},[e]),p=Ae.useCallback(()=>{var j;const L=l.current;if(!L)return;const N=(j=L.error)==null?void 0:j.code;if(N===3&&h.current<2){h.current++,d("loading"),setTimeout(()=>{const ne=l.current;ne!=null&&ne.src&&ne.load()},3e3);return}const B=L.error?`MEDIA_${N}: ${L.error.message}`:"视频加载失败";E(B),d("error")},[]),[T,P]=Ae.useState({scale:1,x:0,y:0}),[S,C]=Ae.useState(!1),y=Ae.useRef(null),I=Ae.useRef({active:!1,startX:0,startY:0,tx:0,ty:0}),v=Ae.useCallback(L=>(C(!0),y.current&&clearTimeout(y.current),y.current=setTimeout(()=>C(!1),1e3),L),[]),A=Ae.useCallback(L=>{L.preventDefault();const N=L.deltaY>0?-.15:.15;P(B=>{const j=Math.max(.5,Math.min(8,B.scale+N));return v(j),{scale:j,x:B.x,y:B.y}})},[v]),F=Ae.useCallback(L=>{L.button===0&&(P(N=>(N.scale<=1||(I.current={active:!0,startX:L.clientX,startY:L.clientY,tx:N.x,ty:N.y}),N)),L.preventDefault())},[]);Ae.useEffect(()=>{const L=B=>{if(!I.current.active)return;const j=B.clientX-I.current.startX,ne=B.clientY-I.current.startY;P(ae=>({...ae,x:I.current.tx+j,y:I.current.ty+ne}))},N=()=>{I.current.active=!1};return window.addEventListener("mousemove",L),window.addEventListener("mouseup",N),()=>{window.removeEventListener("mousemove",L),window.removeEventListener("mouseup",N)}},[]);const D=Ae.useCallback(()=>{P({scale:1,x:0,y:0}),v(1)},[v]);Ae.useEffect(()=>{l.current},[t]),Ae.useEffect(()=>{l.current},[e]);const z=Ae.useCallback(()=>{l.current&&r&&r(l.current.currentTime)},[r]);Ae.useEffect(()=>{const L=o.current;if(!L)return;const N=L.getContext("2d");if(!N)return;const B=L.parentElement;if(B){const re=B.clientWidth,le=B.clientHeight;(re!==f.current.w||le!==f.current.h)&&(L.width=re,L.height=le,f.current={w:re,h:le})}N.clearRect(0,0,L.width,L.height);for(const re of s){const le=re.x*L.width,Ze=re.y*L.height,Ke=re.width*L.width,Q=re.height*L.height,ue=le-Ke/2,he=Ze-Q/2,Be=re.color||"#00ff88";N.fillStyle="rgba(0,255,136,0.08)",N.fillRect(ue,he,Ke,Q),N.strokeStyle=Be,N.lineWidth=2,N.strokeRect(ue,he,Ke,Q);const Ue=`${re.label}${re.confidence?` ${Math.round(re.confidence*100)}%`:""}`;N.font="bold 11px monospace";const ze=N.measureText(Ue).width+12;N.fillStyle=Be,N.fillRect(ue,Math.max(0,he-20),ze,20),N.fillStyle="#000",N.textAlign="left",N.fillText(Ue,ue+6,Math.max(0,he-20)+14)}if(a){const re=a.x*L.width,le=a.y*L.height,Ze=12;N.beginPath(),N.arc(re,le,Ze,0,Math.PI*2),N.strokeStyle="rgba(255, 60, 60, 0.7)",N.lineWidth=2,N.stroke(),N.beginPath(),N.moveTo(re-Ze,le),N.lineTo(re-4,le),N.moveTo(re+4,le),N.lineTo(re+Ze,le),N.moveTo(re,le-Ze),N.lineTo(re,le-4),N.moveTo(re,le+4),N.lineTo(re,le+Ze),N.strokeStyle="rgba(255, 60, 60, 0.9)",N.lineWidth=1.5,N.stroke(),N.beginPath(),N.arc(re,le,2.5,0,Math.PI*2),N.fillStyle="rgba(255, 0, 0, 0.9)",N.fill()}const j=Math.floor(e/60),ne=Math.floor(e%60),ae=`${j.toString().padStart(2,"0")}:${ne.toString().padStart(2,"0")}`;N.fillStyle="rgba(0,0,0,0.6)",N.fillRect(8,L.height-28,82,22),N.fillStyle="rgba(255,255,255,0.8)",N.font="12px monospace",N.textAlign="left",N.fillText(`🎬 ${ae}`,14,L.height-12);const fe=(e*30).toFixed(0);N.fillStyle="rgba(0,0,0,0.6)",N.fillRect(L.width-70,L.height-28,62,22),N.fillStyle="rgba(255,255,255,0.5)",N.textAlign="right",N.fillText(`#${fe}`,L.width-12,L.height-12)},[e,s,a]);const O=T.scale>1?I.current.active?"grabbing":"grab":"default",k=se.jsxs("div",{"aria-hidden":"true",ref:c,onWheel:A,onMouseDown:F,onDoubleClick:D,style:{position:"absolute",inset:0,overflow:"hidden",cursor:O},children:[se.jsxs("div",{style:{position:"absolute",inset:0,transform:`scale(${T.scale}) translate(${T.x/T.scale}px, ${T.y/T.scale}px)`,transformOrigin:"center center",transition:I.current.active?"none":"transform 0.15s ease-out"},children:[se.jsx("video",{ref:l,src:i,style:{width:"100%",height:"100%",objectFit:n},preload:"auto",playsInline:!0,muted:!0,onTimeUpdate:z,onCanPlay:()=>m("canplay"),onPlaying:()=>m("playing"),onWaiting:()=>m("waiting"),onStalled:()=>m("stalled"),onError:p}),se.jsx("canvas",{ref:o,style:{position:"absolute",inset:0,width:"100%",height:"100%",pointerEvents:"none",zIndex:5}})]}),S&&se.jsxs("div",{style:{position:"absolute",top:"50%",left:"50%",transform:"translate(-50%, -50%)",background:"rgba(0,0,0,0.75)",color:"#fff",padding:"4px 14px",borderRadius:20,fontSize:13,fontFamily:"var(--font-mono)",pointerEvents:"none",zIndex:99,transition:"opacity 0.3s"},children:[Math.round(T.scale*100),"%"]})]});return se.jsxs("div",{style:{position:"relative",width:"100%",height:"100%",background:"#000"},children:[k,u==="loading"&&se.jsxs("div",{style:{position:"absolute",inset:0,zIndex:100,display:"flex",flexDirection:"column",alignItems:"center",justifyContent:"center",background:"rgba(0,0,0,0.5)",gap:8,transition:"opacity 0.3s"},children:[se.jsx(Gp,{style:{color:"rgba(255,255,255,0.7)",fontSize:28}}),se.jsx("span",{style:{color:"rgba(255,255,255,0.6)",fontSize:12,fontFamily:"var(--font-mono)"},children:"加载视频流…"})]}),u==="error"&&se.jsxs("div",{style:{position:"absolute",inset:0,zIndex:100,display:"flex",flexDirection:"column",alignItems:"center",justifyContent:"center",background:"rgba(0,0,0,0.7)",gap:6,padding:20},children:[se.jsx("span",{style:{fontSize:24},children:"⚠️"}),se.jsx("span",{style:{color:"rgba(255,255,255,0.7)",fontSize:11,fontFamily:"var(--font-mono)",textAlign:"center",lineHeight:1.4},children:x||"视频无法播放"})]})]})}function Gy({channels:i,activeTopics:e,getVideoSrc:t,currentTime:n,playing:r,coverMode:s,onTimeUpdate:a,gazePoint:o}){const l=i.filter(y=>e.includes(y.topic)),[c,f]=Ae.useState(null),[h,u]=Ae.useState([]),[d,x]=Ae.useState(null),E=(c?l.filter(y=>y.topic===c):l).slice().sort((y,I)=>{const v=h.indexOf(y.topic),A=h.indexOf(I.topic);return v===-1&&A===-1?0:v===-1?1:A===-1?-1:v-A}),m=E.length,p=Ae.useCallback((y,I)=>{y.dataTransfer.setData("text/plain",I),y.dataTransfer.effectAllowed="move"},[]),T=Ae.useCallback((y,I)=>{y.preventDefault(),y.dataTransfer.dropEffect="move",x(I)},[]),P=Ae.useCallback(()=>{x(null)},[]),S=Ae.useCallback((y,I)=>{y.preventDefault(),x(null);const v=y.dataTransfer.getData("text/plain");!v||v===I||u(A=>{const F=l.map(N=>N.topic),D=F.indexOf(v),z=F.indexOf(I);if(D===-1||z===-1)return A;const O=[...A.length?A:F],k=O.indexOf(v),L=O.indexOf(I);return k!==-1&&L!==-1&&(O.splice(k,1),O.splice(L,0,v)),O})},[l]),C=()=>m<=1?{gridTemplateColumns:"1fr"}:m<=2?{gridTemplateColumns:"1fr 1fr"}:m===3?{gridTemplateColumns:"2fr 1fr"}:m<=4?{gridTemplateColumns:"1fr 1fr"}:{gridTemplateColumns:"repeat(3, 1fr)"};return se.jsx("div",{style:{flex:1,padding:8,overflow:"auto",background:"var(--gray-50)",display:"flex"},children:m===0?se.jsx("div",{style:{flex:1,display:"flex",alignItems:"center",justifyContent:"center",color:"var(--gray-400)"},children:"请勾选需要查看的摄像头通道"}):se.jsx("div",{style:{display:"grid",gap:8,width:"100%",...C()},children:E.map((y,I)=>{const v=m===3,A=v&&I===0?"1 / 3":void 0,F=v&&I===0?"1 / 2":v?"2 / 3":void 0,D=d===y.topic;return se.jsxs("div",{style:{display:"flex",flexDirection:"column",background:"#000",borderRadius:"var(--radius-md)",overflow:"hidden",border:D?"2px dashed var(--color-primary)":"1px solid var(--gray-200)",gridRow:A,gridColumn:F,minHeight:m<=1?0:250,opacity:D?.85:1,transition:"border 150ms, opacity 150ms"},children:[se.jsxs("fieldset",{draggable:!0,onDragStart:z=>p(z,y.topic),onDragOver:z=>T(z,y.topic),onDragLeave:P,onDrop:z=>S(z,y.topic),style:{height:28,display:"flex",alignItems:"center",padding:"0 8px",gap:6,background:"rgba(0,0,0,0.5)",borderBottom:"1px solid rgba(255,255,255,0.1)",flexShrink:0,cursor:"grab"},children:[se.jsx("span",{style:{color:"rgba(255,255,255,0.4)",fontSize:14},children:"⠿"}),se.jsx("span",{style:{color:"rgba(255,255,255,0.7)",fontSize:"var(--font-size-xs)",flex:1},children:y.label}),se.jsx(pr,{title:c?"恢复多视图":"放大",children:se.jsx(Ci,{type:"text",size:"small",icon:c?se.jsx(Hp,{style:{color:"rgba(255,255,255,0.5)"}}):se.jsx(Hh,{style:{color:"rgba(255,255,255,0.5)"}}),onClick:()=>f(c===y.topic?null:y.topic),style:{width:24,height:24}})})]}),se.jsx("div",{style:{flex:1,position:"relative"},children:se.jsx(Vy,{src:t(y),currentTime:n,playing:r,objectFit:s?"cover":"contain",onTimeUpdate:a,gazePoint:o})})]},y.topic)})})})}function Hy(){if(typeof document>"u"||!document.cookie)return null;for(const i of document.cookie.split(";")){const[e,...t]=i.trim().split("=");if(e!=="databrew_session")continue;const n=t.join("=");if(!n)return null;try{return decodeURIComponent(n)}catch{return n}}return null}const Gl="";function Wy(i){const e=i.split("/");for(let t=0;t<e.length;t++)if(e[t]==="camera"&&t+1<e.length)return e[t+1];return i.split("/").filter(Boolean).pop()||i}function Xy(i){const e=i.window;if(e!=null&&e.duration_ms&&e.duration_ms>0)return e.duration_ms;const t=i.stats;return t!=null&&t.window_effective_end_ns&&(t!=null&&t.window_effective_start_ns)&&t.window_effective_end_ns>t.window_effective_start_ns?(t.window_effective_end_ns-t.window_effective_start_ns)/1e6:0}function Ky(){var D,z;const[i]=Cp(),[e,t]=Ae.useState(null),[n,r]=Ae.useState(!1),[s,a]=Ae.useState([]),[o,l]=Ae.useState(0),[c,f]=Ae.useState(!1),[h,u]=Ae.useState(!0),[d,x]=Ae.useState(null),E=Ae.useCallback(async O=>{r(!0);try{const k=await fetch(`${Gl}/api/v1/preview/assets/${O}/manifest`,{credentials:"include"});if(!k.ok){Nr.error(`加载失败: ${k.status}`);return}const L=await k.json(),N=L.sources||[],B=L.source_type==="mp4",j=N.filter(le=>le.topic||B);if(j.length===0){Nr.error("该资产没有可预览的视频通道");return}const ne=j.map(le=>({topic:le.topic||le.id,label:le.topic?Wy(le.topic):"video"})),ae=L.mcap,fe=Xy(L);t({assetId:O,channels:ne,sources:j,durationMs:fe,fileSize:ae==null?void 0:ae.size_bytes,rawManifest:L}),a(ne.filter(le=>!le.topic.toLowerCase().includes("side")).map(le=>le.topic)),l(0),f(!1);const re=fe/1e3;x({startSec:0,endSec:Math.min(120,Math.max(re,10))}),fetch(`${Gl}/api/v1/preview/assets/${O}/prewarm`,{method:"POST",credentials:"include"}).catch(()=>{})}catch(k){Nr.error(`请求失败: ${String(k)}`)}finally{r(!1)}},[]),m=Ae.useCallback(O=>{a(k=>k.includes(O)?k.filter(L=>L!==O):[...k,O])},[]),p=Ae.useCallback(O=>{const k=e==null?void 0:e.sources.find(fe=>fe.topic===O.topic||fe.id===O.topic);if(!k)return"";const L=`${Gl}${k.url}`,N=L.indexOf("?"),B=N>=0?L.slice(0,N):L,j=new URLSearchParams(N>=0?L.slice(N+1):"");d&&(j.set("start_sec",String(d.startSec)),j.set("end_sec",String(d.endSec)));const ne=Hy();ne&&j.set("databrew_token",encodeURIComponent(ne));const ae=j.toString();return ae?`${B}?${ae}`:B},[e==null?void 0:e.sources,d]),T=Ae.useRef(null),P=Ae.useRef(0),S=Ae.useCallback(O=>{const k=T.current;if(k){if(performance.now()<k.deadline&&Math.abs(O-k.time)>.5)return;T.current=null,P.current=0}const L=performance.now();L-P.current<90||(P.current=L,l(O))},[]),C=Ae.useMemo(()=>e?{durationMs:e.durationMs,fileSize:e.fileSize,channelCount:e.channels.length,assetId:e.assetId}:null,[e]),y=(D=e==null?void 0:e.rawManifest)==null?void 0:D.stats,I=(z=e==null?void 0:e.rawManifest)==null?void 0:z.window,v=(I==null?void 0:I.start_timestamp_ns)||(y==null?void 0:y.window_effective_start_ns)||0;Ae.useEffect(()=>{const O=i.get("asset");O&&E(O).then(()=>{const k=Number(i.get("start")),L=Number(i.get("end"));k>0&&L>k&&x({startSec:k,endSec:L})})},[i.get,E]);const A=Ae.useCallback(O=>{a(O.activeTopics),u(O.coverMode)},[]),F=Ae.useRef(0);return F.current=e?e.durationMs/1e3:0,Ae.useEffect(()=>{const O=k=>{if(k.target instanceof HTMLInputElement||k.target instanceof HTMLTextAreaElement)return;const L=N=>{document.querySelectorAll("video").forEach(B=>{B.currentTime=N})};switch(k.code){case"Space":k.preventDefault(),f(N=>{const B=!N;return document.querySelectorAll("video").forEach(j=>{B?j.play().catch(()=>{}):j.pause()}),B});break;case"ArrowLeft":k.preventDefault(),l(N=>{const B=Math.max(0,N-.03333333333333333);return L(B),B});break;case"ArrowRight":k.preventDefault(),l(N=>{const B=Math.min(F.current,N+.03333333333333333);return L(B),B});break}};return window.addEventListener("keydown",O),()=>window.removeEventListener("keydown",O)},[]),Ae.useEffect(()=>{if(d){const O=d.startSec;l(O),document.querySelectorAll("video").forEach(k=>{k.currentTime=O})}},[d]),se.jsxs("div",{style:{display:"flex",flexDirection:"column",height:"100%"},children:[se.jsxs("div",{style:{display:"flex",alignItems:"center",gap:4,padding:"4px 12px",background:"#fff",borderBottom:"1px solid var(--gray-100)"},children:[se.jsx("div",{style:{flex:1},children:se.jsx(Wp,{onLoad:E,loading:n,range:d,onRangeChange:x,startTimestampNs:v})}),se.jsx(Xp,{activeTopics:(e==null?void 0:e.channels.filter(O=>s.includes(O.topic)).map(O=>O.topic))||s,coverMode:h,onImport:A})]}),e?se.jsx(Yp,{viewport:se.jsx(Gy,{channels:e.channels,activeTopics:s,getVideoSrc:p,currentTime:o,playing:c,coverMode:h,onTimeUpdate:S}),sidebar:C&&se.jsx(Iy,{channels:e.channels,activeTopics:s,onToggle:m,metadata:C}),timeline:se.jsx(zy,{currentTime:o,durationSec:e.durationMs/1e3,playbackRate:1,playing:c,coverMode:h,onToggleCover:()=>u(O=>!O),range:d?{startSec:d.startSec,endSec:d.endSec,color:"rgba(0, 180, 216, 0.35)",label:"区间"}:void 0,onRangeChange:O=>x(O),startTimestampNs:v,onPlayPause:()=>{f(O=>{const k=!O;return document.querySelectorAll("video").forEach(L=>{k?L.play().catch(()=>{}):L.pause()}),k})},onSeek:O=>{const k=Math.max(0,O);l(k),document.querySelectorAll("video").forEach(L=>{L.currentTime=k}),T.current={time:k,deadline:performance.now()+1e4}},onChangeRate:O=>{document.querySelectorAll("video").forEach(k=>{k.playbackRate=O})}})}):se.jsxs("div",{style:{flex:1,display:"flex",flexDirection:"column",alignItems:"center",justifyContent:"center",gap:12,color:"var(--gray-400)"},children:[se.jsx("span",{style:{fontSize:40},children:"🎥"}),se.jsx("span",{style:{fontSize:"var(--font-size-base)"},children:"输入资产 ID 开始预览"})]})]})}export{Ky as default};
//# sourceMappingURL=PreviewPage-DAJKTpr5.js.map

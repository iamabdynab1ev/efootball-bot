"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[434],{8735:function(e,n,t){t.d(n,{kZ:function(){return i},rg:function(){return o}});var l=t(2265);let u=(0,l.createContext)(null);function o(e){let{clientId:n,nonce:t,onScriptLoadSuccess:o,onScriptLoadError:r,children:i}=e,c=function(){let e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{},{nonce:n,onScriptLoadSuccess:t,onScriptLoadError:u}=e,[o,r]=(0,l.useState)(!1),i=(0,l.useRef)(t);i.current=t;let c=(0,l.useRef)(u);return c.current=u,(0,l.useEffect)(()=>{let e=document.createElement("script");return e.src="https://accounts.google.com/gsi/client",e.async=!0,e.defer=!0,e.nonce=n,e.onload=()=>{var e;r(!0),null===(e=i.current)||void 0===e||e.call(i)},e.onerror=()=>{var e;r(!1),null===(e=c.current)||void 0===e||e.call(c)},document.body.appendChild(e),()=>{document.body.removeChild(e)}},[n]),o}({nonce:t,onScriptLoadSuccess:o,onScriptLoadError:r}),d=(0,l.useMemo)(()=>({clientId:n,scriptLoadedSuccessfully:c}),[n,c]);return l.createElement(u.Provider,{value:d},i)}let r={large:40,medium:32,small:20};function i(e){let{onSuccess:n,onError:t,useOneTap:o,promptMomentNotification:i,type:c="standard",theme:d="outline",size:a="large",text:v,shape:s,logo_alignment:h,width:f,locale:y,click_listener:k,containerProps:p,...m}=e,w=(0,l.useRef)(null),{clientId:g,scriptLoadedSuccessfully:Z}=function(){let e=(0,l.useContext)(u);if(!e)throw Error("Google OAuth components must be used within GoogleOAuthProvider");return e}(),M=(0,l.useRef)(n);M.current=n;let C=(0,l.useRef)(t);C.current=t;let x=(0,l.useRef)(i);return x.current=i,(0,l.useEffect)(()=>{var e,n,t,l,u,r,i,p,E;if(Z)return null===(t=null===(n=null===(e=null==window?void 0:window.google)||void 0===e?void 0:e.accounts)||void 0===n?void 0:n.id)||void 0===t||t.initialize({client_id:g,callback:e=>{var n,t;if(!(null==e?void 0:e.credential))return null===(n=C.current)||void 0===n?void 0:n.call(C);let{credential:l,select_by:u}=e;M.current({credential:l,clientId:null!==(t=null==e?void 0:e.clientId)&&void 0!==t?t:null==e?void 0:e.client_id,select_by:u})},...m}),null===(r=null===(u=null===(l=null==window?void 0:window.google)||void 0===l?void 0:l.accounts)||void 0===u?void 0:u.id)||void 0===r||r.renderButton(w.current,{type:c,theme:d,size:a,text:v,shape:s,logo_alignment:h,width:f,locale:y,click_listener:k}),o&&(null===(E=null===(p=null===(i=null==window?void 0:window.google)||void 0===i?void 0:i.accounts)||void 0===p?void 0:p.id)||void 0===E||E.prompt(x.current)),()=>{var e,n,t;o&&(null===(t=null===(n=null===(e=null==window?void 0:window.google)||void 0===e?void 0:e.accounts)||void 0===n?void 0:n.id)||void 0===t||t.cancel())}},[g,Z,o,c,d,a,v,s,h,f,y]),l.createElement("div",{...p,ref:w,style:{height:r[a],...null==p?void 0:p.style}})}},7019:function(e,n,t){t.d(n,{Z:function(){return l}});/**
 * @license lucide-react v0.414.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */let l=(0,t(8030).Z)("EyeOff",[["path",{d:"M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49",key:"ct8e1f"}],["path",{d:"M14.084 14.158a3 3 0 0 1-4.242-4.242",key:"151rxh"}],["path",{d:"M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143",key:"13bj9a"}],["path",{d:"m2 2 20 20",key:"1ooewy"}]])},5733:function(e,n,t){t.d(n,{Z:function(){return l}});/**
 * @license lucide-react v0.414.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */let l=(0,t(8030).Z)("Eye",[["path",{d:"M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0",key:"1nclc0"}],["circle",{cx:"12",cy:"12",r:"3",key:"1v7zrd"}]])},7017:function(e,n,t){t.d(n,{Z:function(){return l}});/**
 * @license lucide-react v0.414.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */let l=(0,t(8030).Z)("FlaskConical",[["path",{d:"M10 2v7.527a2 2 0 0 1-.211.896L4.72 20.55a1 1 0 0 0 .9 1.45h12.76a1 1 0 0 0 .9-1.45l-5.069-10.127A2 2 0 0 1 14 9.527V2",key:"pzvekw"}],["path",{d:"M8.5 2h7",key:"csnxdl"}],["path",{d:"M7 16h10",key:"wp8him"}]])},9178:function(e,n,t){t.d(n,{Z:function(){return l}});/**
 * @license lucide-react v0.414.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */let l=(0,t(8030).Z)("KeyRound",[["path",{d:"M2.586 17.414A2 2 0 0 0 2 18.828V21a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h1a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h.172a2 2 0 0 0 1.414-.586l.814-.814a6.5 6.5 0 1 0-4-4z",key:"1s6t7t"}],["circle",{cx:"16.5",cy:"7.5",r:".5",fill:"currentColor",key:"w0ekpg"}]])},7905:function(e,n,t){t.d(n,{Z:function(){return l}});/**
 * @license lucide-react v0.414.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */let l=(0,t(8030).Z)("LogIn",[["path",{d:"M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4",key:"u53s6r"}],["polyline",{points:"10 17 15 12 10 7",key:"1ail0h"}],["line",{x1:"15",x2:"3",y1:"12",y2:"12",key:"v6grx8"}]])},6141:function(e,n,t){t.d(n,{Z:function(){return l}});/**
 * @license lucide-react v0.414.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */let l=(0,t(8030).Z)("ShieldCheck",[["path",{d:"M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z",key:"oel41y"}],["path",{d:"m9 12 2 2 4-4",key:"dzmm74"}]])},500:function(e,n,t){t.d(n,{Z:function(){return l}});/**
 * @license lucide-react v0.414.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */let l=(0,t(8030).Z)("Shield",[["path",{d:"M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z",key:"oel41y"}]])},1240:function(e,n,t){t.d(n,{Z:function(){return l}});/**
 * @license lucide-react v0.414.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */let l=(0,t(8030).Z)("Users",[["path",{d:"M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2",key:"1yyitq"}],["circle",{cx:"9",cy:"7",r:"4",key:"nufk8"}],["path",{d:"M22 21v-2a4 4 0 0 0-3-3.87",key:"kshegd"}],["path",{d:"M16 3.13a4 4 0 0 1 0 7.75",key:"1da9ce"}]])},5430:function(e,n,t){t.d(n,{Z:function(){return l}});/**
 * @license lucide-react v0.414.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */let l=(0,t(8030).Z)("Zap",[["path",{d:"M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z",key:"1xq2db"}]])},6463:function(e,n,t){var l=t(1169);t.o(l,"usePathname")&&t.d(n,{usePathname:function(){return l.usePathname}}),t.o(l,"useRouter")&&t.d(n,{useRouter:function(){return l.useRouter}}),t.o(l,"useSearchParams")&&t.d(n,{useSearchParams:function(){return l.useSearchParams}})}}]);
import {
	R as En,
	b as N,
	o as qr,
	k as Rt,
	f as Te,
	c as Ur,
} from "./index-DXdSO-bO.js";
const $e = ["http", "https", "mailto", "tel"];
function Vr(n) {
	const e = (n || "").trim(),
		t = e.charAt(0);
	if (t === "#" || t === "/") return e;
	const r = e.indexOf(":");
	if (r === -1) return e;
	let i = -1;
	for (; ++i < $e.length; ) {
		const l = $e[i];
		if (r === l.length && e.slice(0, l.length).toLowerCase() === l) return e;
	}
	return (
		(i = e.indexOf("?")),
		(i !== -1 && r > i) || ((i = e.indexOf("#")), i !== -1 && r > i)
			? e
			: "javascript:void(0)"
	);
} /*!
 * Determine if an object is a Buffer
 *
 * @author   Feross Aboukhadijeh <https://feross.org>
 * @license  MIT
 */
var re, We;
function $r() {
	return (
		We ||
			((We = 1),
			(re = (e) =>
				e != null &&
				e.constructor != null &&
				typeof e.constructor.isBuffer == "function" &&
				e.constructor.isBuffer(e))),
		re
	);
}
var Wr = $r();
const vt = Te(Wr);
function Nn(n) {
	return !n || typeof n != "object"
		? ""
		: "position" in n || "type" in n
			? Qe(n.position)
			: "start" in n || "end" in n
				? Qe(n)
				: "line" in n || "column" in n
					? be(n)
					: "";
}
function be(n) {
	return Xe(n && n.line) + ":" + Xe(n && n.column);
}
function Qe(n) {
	return be(n && n.start) + "-" + be(n && n.end);
}
function Xe(n) {
	return n && typeof n == "number" ? n : 1;
}
class ln extends Error {
	constructor(e, t, r) {
		const i = [null, null];
		let l = {
			start: { line: null, column: null },
			end: { line: null, column: null },
		};
		if (
			(super(),
			typeof t == "string" && ((r = t), (t = void 0)),
			typeof r == "string")
		) {
			const u = r.indexOf(":");
			u === -1 ? (i[1] = r) : ((i[0] = r.slice(0, u)), (i[1] = r.slice(u + 1)));
		}
		t &&
			("type" in t || "position" in t
				? t.position && (l = t.position)
				: "start" in t || "end" in t
					? (l = t)
					: ("line" in t || "column" in t) && (l.start = t)),
			(this.name = Nn(t) || "1:1"),
			(this.message = typeof e == "object" ? e.message : e),
			(this.stack = ""),
			typeof e == "object" && e.stack && (this.stack = e.stack),
			(this.reason = this.message),
			this.fatal,
			(this.line = l.start.line),
			(this.column = l.start.column),
			(this.position = l),
			(this.source = i[0]),
			(this.ruleId = i[1]),
			this.file,
			this.actual,
			this.expected,
			this.url,
			this.note;
	}
}
ln.prototype.file = "";
ln.prototype.name = "";
ln.prototype.reason = "";
ln.prototype.message = "";
ln.prototype.stack = "";
ln.prototype.fatal = null;
ln.prototype.column = null;
ln.prototype.line = null;
ln.prototype.source = null;
ln.prototype.ruleId = null;
ln.prototype.position = null;
const hn = { basename: Qr, dirname: Xr, extname: Gr, join: Yr, sep: "/" };
function Qr(n, e) {
	if (e !== void 0 && typeof e != "string")
		throw new TypeError('"ext" argument must be a string');
	jn(n);
	let t = 0,
		r = -1,
		i = n.length,
		l;
	if (e === void 0 || e.length === 0 || e.length > n.length) {
		for (; i--; )
			if (n.charCodeAt(i) === 47) {
				if (l) {
					t = i + 1;
					break;
				}
			} else r < 0 && ((l = !0), (r = i + 1));
		return r < 0 ? "" : n.slice(t, r);
	}
	if (e === n) return "";
	let u = -1,
		o = e.length - 1;
	for (; i--; )
		if (n.charCodeAt(i) === 47) {
			if (l) {
				t = i + 1;
				break;
			}
		} else
			u < 0 && ((l = !0), (u = i + 1)),
				o > -1 &&
					(n.charCodeAt(i) === e.charCodeAt(o--)
						? o < 0 && (r = i)
						: ((o = -1), (r = u)));
	return t === r ? (r = u) : r < 0 && (r = n.length), n.slice(t, r);
}
function Xr(n) {
	if ((jn(n), n.length === 0)) return ".";
	let e = -1,
		t = n.length,
		r;
	for (; --t; )
		if (n.charCodeAt(t) === 47) {
			if (r) {
				e = t;
				break;
			}
		} else r || (r = !0);
	return e < 0
		? n.charCodeAt(0) === 47
			? "/"
			: "."
		: e === 1 && n.charCodeAt(0) === 47
			? "//"
			: n.slice(0, e);
}
function Gr(n) {
	jn(n);
	let e = n.length,
		t = -1,
		r = 0,
		i = -1,
		l = 0,
		u;
	for (; e--; ) {
		const o = n.charCodeAt(e);
		if (o === 47) {
			if (u) {
				r = e + 1;
				break;
			}
			continue;
		}
		t < 0 && ((u = !0), (t = e + 1)),
			o === 46 ? (i < 0 ? (i = e) : l !== 1 && (l = 1)) : i > -1 && (l = -1);
	}
	return i < 0 || t < 0 || l === 0 || (l === 1 && i === t - 1 && i === r + 1)
		? ""
		: n.slice(i, t);
}
function Yr(...n) {
	let e = -1,
		t;
	for (; ++e < n.length; )
		jn(n[e]), n[e] && (t = t === void 0 ? n[e] : t + "/" + n[e]);
	return t === void 0 ? "." : Kr(t);
}
function Kr(n) {
	jn(n);
	const e = n.charCodeAt(0) === 47;
	let t = Zr(n, !e);
	return (
		t.length === 0 && !e && (t = "."),
		t.length > 0 && n.charCodeAt(n.length - 1) === 47 && (t += "/"),
		e ? "/" + t : t
	);
}
function Zr(n, e) {
	let t = "",
		r = 0,
		i = -1,
		l = 0,
		u = -1,
		o,
		a;
	for (; ++u <= n.length; ) {
		if (u < n.length) o = n.charCodeAt(u);
		else {
			if (o === 47) break;
			o = 47;
		}
		if (o === 47) {
			if (!(i === u - 1 || l === 1))
				if (i !== u - 1 && l === 2) {
					if (
						t.length < 2 ||
						r !== 2 ||
						t.charCodeAt(t.length - 1) !== 46 ||
						t.charCodeAt(t.length - 2) !== 46
					) {
						if (t.length > 2) {
							if (((a = t.lastIndexOf("/")), a !== t.length - 1)) {
								a < 0
									? ((t = ""), (r = 0))
									: ((t = t.slice(0, a)),
										(r = t.length - 1 - t.lastIndexOf("/"))),
									(i = u),
									(l = 0);
								continue;
							}
						} else if (t.length > 0) {
							(t = ""), (r = 0), (i = u), (l = 0);
							continue;
						}
					}
					e && ((t = t.length > 0 ? t + "/.." : ".."), (r = 2));
				} else
					t.length > 0
						? (t += "/" + n.slice(i + 1, u))
						: (t = n.slice(i + 1, u)),
						(r = u - i - 1);
			(i = u), (l = 0);
		} else o === 46 && l > -1 ? l++ : (l = -1);
	}
	return t;
}
function jn(n) {
	if (typeof n != "string")
		throw new TypeError("Path must be a string. Received " + JSON.stringify(n));
}
const Jr = { cwd: ni };
function ni() {
	return "/";
}
function we(n) {
	return n !== null && typeof n == "object" && n.href && n.origin;
}
function ei(n) {
	if (typeof n == "string") n = new URL(n);
	else if (!we(n)) {
		const e = new TypeError(
			'The "path" argument must be of type string or an instance of URL. Received `' +
				n +
				"`",
		);
		throw ((e.code = "ERR_INVALID_ARG_TYPE"), e);
	}
	if (n.protocol !== "file:") {
		const e = new TypeError("The URL must be of scheme file");
		throw ((e.code = "ERR_INVALID_URL_SCHEME"), e);
	}
	return ti(n);
}
function ti(n) {
	if (n.hostname !== "") {
		const r = new TypeError(
			'File URL host must be "localhost" or empty on darwin',
		);
		throw ((r.code = "ERR_INVALID_FILE_URL_HOST"), r);
	}
	const e = n.pathname;
	let t = -1;
	for (; ++t < e.length; )
		if (e.charCodeAt(t) === 37 && e.charCodeAt(t + 1) === 50) {
			const r = e.charCodeAt(t + 2);
			if (r === 70 || r === 102) {
				const i = new TypeError(
					"File URL path must not include encoded / characters",
				);
				throw ((i.code = "ERR_INVALID_FILE_URL_PATH"), i);
			}
		}
	return decodeURIComponent(e);
}
const ie = ["history", "path", "basename", "stem", "extname", "dirname"];
class Bt {
	constructor(e) {
		let t;
		e
			? typeof e == "string" || ri(e)
				? (t = { value: e })
				: we(e)
					? (t = { path: e })
					: (t = e)
			: (t = {}),
			(this.data = {}),
			(this.messages = []),
			(this.history = []),
			(this.cwd = Jr.cwd()),
			this.value,
			this.stored,
			this.result,
			this.map;
		let r = -1;
		for (; ++r < ie.length; ) {
			const l = ie[r];
			l in t &&
				t[l] !== void 0 &&
				t[l] !== null &&
				(this[l] = l === "history" ? [...t[l]] : t[l]);
		}
		let i;
		for (i in t) ie.includes(i) || (this[i] = t[i]);
	}
	get path() {
		return this.history[this.history.length - 1];
	}
	set path(e) {
		we(e) && (e = ei(e)),
			ue(e, "path"),
			this.path !== e && this.history.push(e);
	}
	get dirname() {
		return typeof this.path == "string" ? hn.dirname(this.path) : void 0;
	}
	set dirname(e) {
		Ge(this.basename, "dirname"), (this.path = hn.join(e || "", this.basename));
	}
	get basename() {
		return typeof this.path == "string" ? hn.basename(this.path) : void 0;
	}
	set basename(e) {
		ue(e, "basename"),
			le(e, "basename"),
			(this.path = hn.join(this.dirname || "", e));
	}
	get extname() {
		return typeof this.path == "string" ? hn.extname(this.path) : void 0;
	}
	set extname(e) {
		if ((le(e, "extname"), Ge(this.dirname, "extname"), e)) {
			if (e.charCodeAt(0) !== 46)
				throw new Error("`extname` must start with `.`");
			if (e.includes(".", 1))
				throw new Error("`extname` cannot contain multiple dots");
		}
		this.path = hn.join(this.dirname, this.stem + (e || ""));
	}
	get stem() {
		return typeof this.path == "string"
			? hn.basename(this.path, this.extname)
			: void 0;
	}
	set stem(e) {
		ue(e, "stem"),
			le(e, "stem"),
			(this.path = hn.join(this.dirname || "", e + (this.extname || "")));
	}
	toString(e) {
		return (this.value || "").toString(e || void 0);
	}
	message(e, t, r) {
		const i = new ln(e, t, r);
		return (
			this.path && ((i.name = this.path + ":" + i.name), (i.file = this.path)),
			(i.fatal = !1),
			this.messages.push(i),
			i
		);
	}
	info(e, t, r) {
		const i = this.message(e, t, r);
		return (i.fatal = null), i;
	}
	fail(e, t, r) {
		const i = this.message(e, t, r);
		throw ((i.fatal = !0), i);
	}
}
function le(n, e) {
	if (n && n.includes(hn.sep))
		throw new Error(
			"`" + e + "` cannot be a path: did not expect `" + hn.sep + "`",
		);
}
function ue(n, e) {
	if (!n) throw new Error("`" + e + "` cannot be empty");
}
function Ge(n, e) {
	if (!n) throw new Error("Setting `" + e + "` requires `path` to be set too");
}
function ri(n) {
	return vt(n);
}
function Ye(n) {
	if (n) throw n;
}
var oe, Ke;
function ii() {
	if (Ke) return oe;
	Ke = 1;
	var n = Object.prototype.hasOwnProperty,
		e = Object.prototype.toString,
		t = Object.defineProperty,
		r = Object.getOwnPropertyDescriptor,
		i = (s) =>
			typeof Array.isArray == "function"
				? Array.isArray(s)
				: e.call(s) === "[object Array]",
		l = (s) => {
			if (!s || e.call(s) !== "[object Object]") return !1;
			var f = n.call(s, "constructor"),
				c =
					s.constructor &&
					s.constructor.prototype &&
					n.call(s.constructor.prototype, "isPrototypeOf");
			if (s.constructor && !f && !c) return !1;
			var p;
			for (p in s);
			return typeof p > "u" || n.call(s, p);
		},
		u = (s, f) => {
			t && f.name === "__proto__"
				? t(s, f.name, {
						enumerable: !0,
						configurable: !0,
						value: f.newValue,
						writable: !0,
					})
				: (s[f.name] = f.newValue);
		},
		o = (s, f) => {
			if (f === "__proto__")
				if (n.call(s, f)) {
					if (r) return r(s, f).value;
				} else return;
			return s[f];
		};
	return (
		(oe = function a() {
			var s,
				f,
				c,
				p,
				h,
				d,
				y = arguments[0],
				S = 1,
				k = arguments.length,
				A = !1;
			for (
				typeof y == "boolean" && ((A = y), (y = arguments[1] || {}), (S = 2)),
					(y == null || (typeof y != "object" && typeof y != "function")) &&
						(y = {});
				S < k;
				++S
			)
				if (((s = arguments[S]), s != null))
					for (f in s)
						(c = o(y, f)),
							(p = o(s, f)),
							y !== p &&
								(A && p && (l(p) || (h = i(p)))
									? (h
											? ((h = !1), (d = c && i(c) ? c : []))
											: (d = c && l(c) ? c : {}),
										u(y, { name: f, newValue: a(A, d, p) }))
									: typeof p < "u" && u(y, { name: f, newValue: p }));
			return y;
		}),
		oe
	);
}
var li = ii();
const Ze = Te(li);
function Se(n) {
	if (typeof n != "object" || n === null) return !1;
	const e = Object.getPrototypeOf(n);
	return (
		(e === null ||
			e === Object.prototype ||
			Object.getPrototypeOf(e) === null) &&
		!(Symbol.toStringTag in n) &&
		!(Symbol.iterator in n)
	);
}
function ui() {
	const n = [],
		e = { run: t, use: r };
	return e;
	function t(...i) {
		let l = -1;
		const u = i.pop();
		if (typeof u != "function")
			throw new TypeError("Expected function as last argument, not " + u);
		o(null, ...i);
		function o(a, ...s) {
			const f = n[++l];
			let c = -1;
			if (a) {
				u(a);
				return;
			}
			for (; ++c < i.length; )
				(s[c] === null || s[c] === void 0) && (s[c] = i[c]);
			(i = s), f ? oi(f, o)(...s) : u(null, ...s);
		}
	}
	function r(i) {
		if (typeof i != "function")
			throw new TypeError("Expected `middelware` to be a function, not " + i);
		return n.push(i), e;
	}
}
function oi(n, e) {
	let t;
	return r;
	function r(...u) {
		const o = n.length > u.length;
		let a;
		o && u.push(i);
		try {
			a = n.apply(this, u);
		} catch (s) {
			const f = s;
			if (o && t) throw f;
			return i(f);
		}
		o ||
			(a instanceof Promise ? a.then(l, i) : a instanceof Error ? i(a) : l(a));
	}
	function i(u, ...o) {
		t || ((t = !0), e(u, ...o));
	}
	function l(u) {
		i(null, u);
	}
}
const ai = Nt().freeze(),
	Mt = {}.hasOwnProperty;
function Nt() {
	const n = ui(),
		e = [];
	let t = {},
		r,
		i = -1;
	return (
		(l.data = u),
		(l.Parser = void 0),
		(l.Compiler = void 0),
		(l.freeze = o),
		(l.attachers = e),
		(l.use = a),
		(l.parse = s),
		(l.stringify = f),
		(l.run = c),
		(l.runSync = p),
		(l.process = h),
		(l.processSync = d),
		l
	);
	function l() {
		const y = Nt();
		let S = -1;
		for (; ++S < e.length; ) y.use(...e[S]);
		return y.data(Ze(!0, {}, t)), y;
	}
	function u(y, S) {
		return typeof y == "string"
			? arguments.length === 2
				? (ce("data", r), (t[y] = S), l)
				: (Mt.call(t, y) && t[y]) || null
			: y
				? (ce("data", r), (t = y), l)
				: t;
	}
	function o() {
		if (r) return l;
		for (; ++i < e.length; ) {
			const [y, ...S] = e[i];
			if (S[0] === !1) continue;
			S[0] === !0 && (S[0] = void 0);
			const k = y.call(l, ...S);
			typeof k == "function" && n.use(k);
		}
		return (r = !0), (i = Number.POSITIVE_INFINITY), l;
	}
	function a(y, ...S) {
		let k;
		if ((ce("use", r), y != null))
			if (typeof y == "function") R(y, ...S);
			else if (typeof y == "object") Array.isArray(y) ? L(y) : C(y);
			else throw new TypeError("Expected usable value, not `" + y + "`");
		return k && (t.settings = Object.assign(t.settings || {}, k)), l;
		function A(x) {
			if (typeof x == "function") R(x);
			else if (typeof x == "object")
				if (Array.isArray(x)) {
					const [D, ...M] = x;
					R(D, ...M);
				} else C(x);
			else throw new TypeError("Expected usable value, not `" + x + "`");
		}
		function C(x) {
			L(x.plugins), x.settings && (k = Object.assign(k || {}, x.settings));
		}
		function L(x) {
			let D = -1;
			if (x != null)
				if (Array.isArray(x))
					for (; ++D < x.length; ) {
						const M = x[D];
						A(M);
					}
				else throw new TypeError("Expected a list of plugins, not `" + x + "`");
		}
		function R(x, D) {
			let M = -1,
				H;
			for (; ++M < e.length; )
				if (e[M][0] === x) {
					H = e[M];
					break;
				}
			H
				? (Se(H[1]) && Se(D) && (D = Ze(!0, H[1], D)), (H[1] = D))
				: e.push([...arguments]);
		}
	}
	function s(y) {
		l.freeze();
		const S = Mn(y),
			k = l.Parser;
		return (
			ae("parse", k),
			Je(k, "parse") ? new k(String(S), S).parse() : k(String(S), S)
		);
	}
	function f(y, S) {
		l.freeze();
		const k = Mn(S),
			A = l.Compiler;
		return (
			se("stringify", A),
			nt(y),
			Je(A, "compile") ? new A(y, k).compile() : A(y, k)
		);
	}
	function c(y, S, k) {
		if (
			(nt(y),
			l.freeze(),
			!k && typeof S == "function" && ((k = S), (S = void 0)),
			!k)
		)
			return new Promise(A);
		A(null, k);
		function A(C, L) {
			n.run(y, Mn(S), R);
			function R(x, D, M) {
				(D = D || y), x ? L(x) : C ? C(D) : k(null, D, M);
			}
		}
	}
	function p(y, S) {
		let k, A;
		return l.run(y, S, C), et("runSync", "run", A), k;
		function C(L, R) {
			Ye(L), (k = R), (A = !0);
		}
	}
	function h(y, S) {
		if ((l.freeze(), ae("process", l.Parser), se("process", l.Compiler), !S))
			return new Promise(k);
		k(null, S);
		function k(A, C) {
			const L = Mn(y);
			l.run(l.parse(L), L, (x, D, M) => {
				if (x || !D || !M) R(x);
				else {
					const H = l.stringify(D, M);
					H == null || (fi(H) ? (M.value = H) : (M.result = H)), R(x, M);
				}
			});
			function R(x, D) {
				x || !D ? C(x) : A ? A(D) : S(null, D);
			}
		}
	}
	function d(y) {
		let S;
		l.freeze(), ae("processSync", l.Parser), se("processSync", l.Compiler);
		const k = Mn(y);
		return l.process(k, A), et("processSync", "process", S), k;
		function A(C) {
			(S = !0), Ye(C);
		}
	}
}
function Je(n, e) {
	return (
		typeof n == "function" &&
		n.prototype &&
		(si(n.prototype) || e in n.prototype)
	);
}
function si(n) {
	let e;
	for (e in n) if (Mt.call(n, e)) return !0;
	return !1;
}
function ae(n, e) {
	if (typeof e != "function")
		throw new TypeError("Cannot `" + n + "` without `Parser`");
}
function se(n, e) {
	if (typeof e != "function")
		throw new TypeError("Cannot `" + n + "` without `Compiler`");
}
function ce(n, e) {
	if (e)
		throw new Error(
			"Cannot call `" +
				n +
				"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.",
		);
}
function nt(n) {
	if (!Se(n) || typeof n.type != "string")
		throw new TypeError("Expected node, got `" + n + "`");
}
function et(n, e, t) {
	if (!t)
		throw new Error("`" + n + "` finished async. Use `" + e + "` instead");
}
function Mn(n) {
	return ci(n) ? n : new Bt(n);
}
function ci(n) {
	return !!(n && typeof n == "object" && "message" in n && "messages" in n);
}
function fi(n) {
	return typeof n == "string" || vt(n);
}
const hi = {};
function pi(n, e) {
	const t = hi,
		r = typeof t.includeImageAlt == "boolean" ? t.includeImageAlt : !0,
		i = typeof t.includeHtml == "boolean" ? t.includeHtml : !0;
	return _t(n, r, i);
}
function _t(n, e, t) {
	if (mi(n)) {
		if ("value" in n) return n.type === "html" && !t ? "" : n.value;
		if (e && "alt" in n && n.alt) return n.alt;
		if ("children" in n) return tt(n.children, e, t);
	}
	return Array.isArray(n) ? tt(n, e, t) : "";
}
function tt(n, e, t) {
	const r = [];
	let i = -1;
	for (; ++i < n.length; ) r[i] = _t(n[i], e, t);
	return r.join("");
}
function mi(n) {
	return !!(n && typeof n == "object");
}
function tn(n, e, t, r) {
	const i = n.length;
	let l = 0,
		u;
	if (
		(e < 0 ? (e = -e > i ? 0 : i + e) : (e = e > i ? i : e),
		(t = t > 0 ? t : 0),
		r.length < 1e4)
	)
		(u = Array.from(r)), u.unshift(e, t), n.splice(...u);
	else
		for (t && n.splice(e, t); l < r.length; )
			(u = r.slice(l, l + 1e4)),
				u.unshift(e, 0),
				n.splice(...u),
				(l += 1e4),
				(e += 1e4);
}
function rn(n, e) {
	return n.length > 0 ? (tn(n, n.length, 0, e), n) : e;
}
const rt = {}.hasOwnProperty;
function jt(n) {
	const e = {};
	let t = -1;
	for (; ++t < n.length; ) gi(e, n[t]);
	return e;
}
function gi(n, e) {
	let t;
	for (t in e) {
		const i = (rt.call(n, t) ? n[t] : void 0) || (n[t] = {}),
			l = e[t];
		let u;
		if (l)
			for (u in l) {
				rt.call(i, u) || (i[u] = []);
				const o = l[u];
				di(i[u], Array.isArray(o) ? o : o ? [o] : []);
			}
	}
}
function di(n, e) {
	let t = -1;
	const r = [];
	for (; ++t < e.length; ) (e[t].add === "after" ? n : r).push(e[t]);
	tn(n, 0, 0, r);
}
const yi =
		/[!-/:-@[-`{-~\xA1\xA7\xAB\xB6\xB7\xBB\xBF\u037E\u0387\u055A-\u055F\u0589\u058A\u05BE\u05C0\u05C3\u05C6\u05F3\u05F4\u0609\u060A\u060C\u060D\u061B\u061D-\u061F\u066A-\u066D\u06D4\u0700-\u070D\u07F7-\u07F9\u0830-\u083E\u085E\u0964\u0965\u0970\u09FD\u0A76\u0AF0\u0C77\u0C84\u0DF4\u0E4F\u0E5A\u0E5B\u0F04-\u0F12\u0F14\u0F3A-\u0F3D\u0F85\u0FD0-\u0FD4\u0FD9\u0FDA\u104A-\u104F\u10FB\u1360-\u1368\u1400\u166E\u169B\u169C\u16EB-\u16ED\u1735\u1736\u17D4-\u17D6\u17D8-\u17DA\u1800-\u180A\u1944\u1945\u1A1E\u1A1F\u1AA0-\u1AA6\u1AA8-\u1AAD\u1B5A-\u1B60\u1B7D\u1B7E\u1BFC-\u1BFF\u1C3B-\u1C3F\u1C7E\u1C7F\u1CC0-\u1CC7\u1CD3\u2010-\u2027\u2030-\u2043\u2045-\u2051\u2053-\u205E\u207D\u207E\u208D\u208E\u2308-\u230B\u2329\u232A\u2768-\u2775\u27C5\u27C6\u27E6-\u27EF\u2983-\u2998\u29D8-\u29DB\u29FC\u29FD\u2CF9-\u2CFC\u2CFE\u2CFF\u2D70\u2E00-\u2E2E\u2E30-\u2E4F\u2E52-\u2E5D\u3001-\u3003\u3008-\u3011\u3014-\u301F\u3030\u303D\u30A0\u30FB\uA4FE\uA4FF\uA60D-\uA60F\uA673\uA67E\uA6F2-\uA6F7\uA874-\uA877\uA8CE\uA8CF\uA8F8-\uA8FA\uA8FC\uA92E\uA92F\uA95F\uA9C1-\uA9CD\uA9DE\uA9DF\uAA5C-\uAA5F\uAADE\uAADF\uAAF0\uAAF1\uABEB\uFD3E\uFD3F\uFE10-\uFE19\uFE30-\uFE52\uFE54-\uFE61\uFE63\uFE68\uFE6A\uFE6B\uFF01-\uFF03\uFF05-\uFF0A\uFF0C-\uFF0F\uFF1A\uFF1B\uFF1F\uFF20\uFF3B-\uFF3D\uFF3F\uFF5B\uFF5D\uFF5F-\uFF65]/,
	J = Sn(/[A-Za-z]/),
	Z = Sn(/[\dA-Za-z]/),
	ki = Sn(/[#-'*+\--9=?A-Z^-~]/);
function Xn(n) {
	return n !== null && (n < 32 || n === 127);
}
const Ce = Sn(/\d/),
	xi = Sn(/[\dA-Fa-f]/),
	bi = Sn(/[!-/:-@[-`{-~]/);
function z(n) {
	return n !== null && n < -2;
}
function $(n) {
	return n !== null && (n < 0 || n === 32);
}
function j(n) {
	return n === -2 || n === -1 || n === 32;
}
const Kn = Sn(yi),
	An = Sn(/\s/);
function Sn(n) {
	return e;
	function e(t) {
		return t !== null && n.test(String.fromCharCode(t));
	}
}
function U(n, e, t, r) {
	const i = r ? r - 1 : Number.POSITIVE_INFINITY;
	let l = 0;
	return u;
	function u(a) {
		return j(a) ? (n.enter(t), o(a)) : e(a);
	}
	function o(a) {
		return j(a) && l++ < i ? (n.consume(a), o) : (n.exit(t), e(a));
	}
}
const wi = { tokenize: Si };
function Si(n) {
	const e = n.attempt(this.parser.constructs.contentInitial, r, i);
	let t;
	return e;
	function r(o) {
		if (o === null) {
			n.consume(o);
			return;
		}
		return (
			n.enter("lineEnding"),
			n.consume(o),
			n.exit("lineEnding"),
			U(n, e, "linePrefix")
		);
	}
	function i(o) {
		return n.enter("paragraph"), l(o);
	}
	function l(o) {
		const a = n.enter("chunkText", { contentType: "text", previous: t });
		return t && (t.next = a), (t = a), u(o);
	}
	function u(o) {
		if (o === null) {
			n.exit("chunkText"), n.exit("paragraph"), n.consume(o);
			return;
		}
		return z(o) ? (n.consume(o), n.exit("chunkText"), l) : (n.consume(o), u);
	}
}
const Ci = { tokenize: Ei },
	it = { tokenize: Ai };
function Ei(n) {
	const e = this,
		t = [];
	let r = 0,
		i,
		l,
		u;
	return o;
	function o(C) {
		if (r < t.length) {
			const L = t[r];
			return (e.containerState = L[1]), n.attempt(L[0].continuation, a, s)(C);
		}
		return s(C);
	}
	function a(C) {
		if ((r++, e.containerState._closeFlow)) {
			(e.containerState._closeFlow = void 0), i && A();
			const L = e.events.length;
			let R = L,
				x;
			for (; R--; )
				if (e.events[R][0] === "exit" && e.events[R][1].type === "chunkFlow") {
					x = e.events[R][1].end;
					break;
				}
			k(r);
			let D = L;
			for (; D < e.events.length; )
				(e.events[D][1].end = Object.assign({}, x)), D++;
			return (
				tn(e.events, R + 1, 0, e.events.slice(L)), (e.events.length = D), s(C)
			);
		}
		return o(C);
	}
	function s(C) {
		if (r === t.length) {
			if (!i) return p(C);
			if (i.currentConstruct && i.currentConstruct.concrete) return d(C);
			e.interrupt = !!(i.currentConstruct && !i._gfmTableDynamicInterruptHack);
		}
		return (e.containerState = {}), n.check(it, f, c)(C);
	}
	function f(C) {
		return i && A(), k(r), p(C);
	}
	function c(C) {
		return (
			(e.parser.lazy[e.now().line] = r !== t.length), (u = e.now().offset), d(C)
		);
	}
	function p(C) {
		return (e.containerState = {}), n.attempt(it, h, d)(C);
	}
	function h(C) {
		return r++, t.push([e.currentConstruct, e.containerState]), p(C);
	}
	function d(C) {
		if (C === null) {
			i && A(), k(0), n.consume(C);
			return;
		}
		return (
			(i = i || e.parser.flow(e.now())),
			n.enter("chunkFlow", { contentType: "flow", previous: l, _tokenizer: i }),
			y(C)
		);
	}
	function y(C) {
		if (C === null) {
			S(n.exit("chunkFlow"), !0), k(0), n.consume(C);
			return;
		}
		return z(C)
			? (n.consume(C),
				S(n.exit("chunkFlow")),
				(r = 0),
				(e.interrupt = void 0),
				o)
			: (n.consume(C), y);
	}
	function S(C, L) {
		const R = e.sliceStream(C);
		if (
			(L && R.push(null),
			(C.previous = l),
			l && (l.next = C),
			(l = C),
			i.defineSkip(C.start),
			i.write(R),
			e.parser.lazy[C.start.line])
		) {
			let x = i.events.length;
			for (; x--; )
				if (
					i.events[x][1].start.offset < u &&
					(!i.events[x][1].end || i.events[x][1].end.offset > u)
				)
					return;
			const D = e.events.length;
			let M = D,
				H,
				b;
			for (; M--; )
				if (e.events[M][0] === "exit" && e.events[M][1].type === "chunkFlow") {
					if (H) {
						b = e.events[M][1].end;
						break;
					}
					H = !0;
				}
			for (k(r), x = D; x < e.events.length; )
				(e.events[x][1].end = Object.assign({}, b)), x++;
			tn(e.events, M + 1, 0, e.events.slice(D)), (e.events.length = x);
		}
	}
	function k(C) {
		let L = t.length;
		for (; L-- > C; ) {
			const R = t[L];
			(e.containerState = R[1]), R[0].exit.call(e, n);
		}
		t.length = C;
	}
	function A() {
		i.write([null]),
			(l = void 0),
			(i = void 0),
			(e.containerState._closeFlow = void 0);
	}
}
function Ai(n, e, t) {
	return U(
		n,
		n.attempt(this.parser.constructs.document, e, t),
		"linePrefix",
		this.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4,
	);
}
function Gn(n) {
	if (n === null || $(n) || An(n)) return 1;
	if (Kn(n)) return 2;
}
function Zn(n, e, t) {
	const r = [];
	let i = -1;
	for (; ++i < n.length; ) {
		const l = n[i].resolveAll;
		l && !r.includes(l) && ((e = l(e, t)), r.push(l));
	}
	return e;
}
const Ee = { name: "attention", tokenize: Di, resolveAll: Fi };
function Fi(n, e) {
	let t = -1,
		r,
		i,
		l,
		u,
		o,
		a,
		s,
		f;
	for (; ++t < n.length; )
		if (
			n[t][0] === "enter" &&
			n[t][1].type === "attentionSequence" &&
			n[t][1]._close
		) {
			for (r = t; r--; )
				if (
					n[r][0] === "exit" &&
					n[r][1].type === "attentionSequence" &&
					n[r][1]._open &&
					e.sliceSerialize(n[r][1]).charCodeAt(0) ===
						e.sliceSerialize(n[t][1]).charCodeAt(0)
				) {
					if (
						(n[r][1]._close || n[t][1]._open) &&
						(n[t][1].end.offset - n[t][1].start.offset) % 3 &&
						!(
							(n[r][1].end.offset -
								n[r][1].start.offset +
								n[t][1].end.offset -
								n[t][1].start.offset) %
							3
						)
					)
						continue;
					a =
						n[r][1].end.offset - n[r][1].start.offset > 1 &&
						n[t][1].end.offset - n[t][1].start.offset > 1
							? 2
							: 1;
					const c = Object.assign({}, n[r][1].end),
						p = Object.assign({}, n[t][1].start);
					lt(c, -a),
						lt(p, a),
						(u = {
							type: a > 1 ? "strongSequence" : "emphasisSequence",
							start: c,
							end: Object.assign({}, n[r][1].end),
						}),
						(o = {
							type: a > 1 ? "strongSequence" : "emphasisSequence",
							start: Object.assign({}, n[t][1].start),
							end: p,
						}),
						(l = {
							type: a > 1 ? "strongText" : "emphasisText",
							start: Object.assign({}, n[r][1].end),
							end: Object.assign({}, n[t][1].start),
						}),
						(i = {
							type: a > 1 ? "strong" : "emphasis",
							start: Object.assign({}, u.start),
							end: Object.assign({}, o.end),
						}),
						(n[r][1].end = Object.assign({}, u.start)),
						(n[t][1].start = Object.assign({}, o.end)),
						(s = []),
						n[r][1].end.offset - n[r][1].start.offset &&
							(s = rn(s, [
								["enter", n[r][1], e],
								["exit", n[r][1], e],
							])),
						(s = rn(s, [
							["enter", i, e],
							["enter", u, e],
							["exit", u, e],
							["enter", l, e],
						])),
						(s = rn(
							s,
							Zn(e.parser.constructs.insideSpan.null, n.slice(r + 1, t), e),
						)),
						(s = rn(s, [
							["exit", l, e],
							["enter", o, e],
							["exit", o, e],
							["exit", i, e],
						])),
						n[t][1].end.offset - n[t][1].start.offset
							? ((f = 2),
								(s = rn(s, [
									["enter", n[t][1], e],
									["exit", n[t][1], e],
								])))
							: (f = 0),
						tn(n, r - 1, t - r + 3, s),
						(t = r + s.length - f - 2);
					break;
				}
		}
	for (t = -1; ++t < n.length; )
		n[t][1].type === "attentionSequence" && (n[t][1].type = "data");
	return n;
}
function Di(n, e) {
	const t = this.parser.constructs.attentionMarkers.null,
		r = this.previous,
		i = Gn(r);
	let l;
	return u;
	function u(a) {
		return (l = a), n.enter("attentionSequence"), o(a);
	}
	function o(a) {
		if (a === l) return n.consume(a), o;
		const s = n.exit("attentionSequence"),
			f = Gn(a),
			c = !f || (f === 2 && i) || t.includes(a),
			p = !i || (i === 2 && f) || t.includes(r);
		return (
			(s._open = !!(l === 42 ? c : c && (i || !p))),
			(s._close = !!(l === 42 ? p : p && (f || !c))),
			e(a)
		);
	}
}
function lt(n, e) {
	(n.column += e), (n.offset += e), (n._bufferIndex += e);
}
const Ii = { name: "autolink", tokenize: Ti };
function Ti(n, e, t) {
	let r = 0;
	return i;
	function i(h) {
		return (
			n.enter("autolink"),
			n.enter("autolinkMarker"),
			n.consume(h),
			n.exit("autolinkMarker"),
			n.enter("autolinkProtocol"),
			l
		);
	}
	function l(h) {
		return J(h) ? (n.consume(h), u) : s(h);
	}
	function u(h) {
		return h === 43 || h === 45 || h === 46 || Z(h) ? ((r = 1), o(h)) : s(h);
	}
	function o(h) {
		return h === 58
			? (n.consume(h), (r = 0), a)
			: (h === 43 || h === 45 || h === 46 || Z(h)) && r++ < 32
				? (n.consume(h), o)
				: ((r = 0), s(h));
	}
	function a(h) {
		return h === 62
			? (n.exit("autolinkProtocol"),
				n.enter("autolinkMarker"),
				n.consume(h),
				n.exit("autolinkMarker"),
				n.exit("autolink"),
				e)
			: h === null || h === 32 || h === 60 || Xn(h)
				? t(h)
				: (n.consume(h), a);
	}
	function s(h) {
		return h === 64 ? (n.consume(h), f) : ki(h) ? (n.consume(h), s) : t(h);
	}
	function f(h) {
		return Z(h) ? c(h) : t(h);
	}
	function c(h) {
		return h === 46
			? (n.consume(h), (r = 0), f)
			: h === 62
				? ((n.exit("autolinkProtocol").type = "autolinkEmail"),
					n.enter("autolinkMarker"),
					n.consume(h),
					n.exit("autolinkMarker"),
					n.exit("autolink"),
					e)
				: p(h);
	}
	function p(h) {
		if ((h === 45 || Z(h)) && r++ < 63) {
			const d = h === 45 ? p : c;
			return n.consume(h), d;
		}
		return t(h);
	}
}
const Hn = { tokenize: Pi, partial: !0 };
function Pi(n, e, t) {
	return r;
	function r(l) {
		return j(l) ? U(n, i, "linePrefix")(l) : i(l);
	}
	function i(l) {
		return l === null || z(l) ? e(l) : t(l);
	}
}
const Ht = {
	name: "blockQuote",
	tokenize: Li,
	continuation: { tokenize: Oi },
	exit: zi,
};
function Li(n, e, t) {
	const r = this;
	return i;
	function i(u) {
		if (u === 62) {
			const o = r.containerState;
			return (
				o.open || (n.enter("blockQuote", { _container: !0 }), (o.open = !0)),
				n.enter("blockQuotePrefix"),
				n.enter("blockQuoteMarker"),
				n.consume(u),
				n.exit("blockQuoteMarker"),
				l
			);
		}
		return t(u);
	}
	function l(u) {
		return j(u)
			? (n.enter("blockQuotePrefixWhitespace"),
				n.consume(u),
				n.exit("blockQuotePrefixWhitespace"),
				n.exit("blockQuotePrefix"),
				e)
			: (n.exit("blockQuotePrefix"), e(u));
	}
}
function Oi(n, e, t) {
	const r = this;
	return i;
	function i(u) {
		return j(u)
			? U(
					n,
					l,
					"linePrefix",
					r.parser.constructs.disable.null.includes("codeIndented")
						? void 0
						: 4,
				)(u)
			: l(u);
	}
	function l(u) {
		return n.attempt(Ht, e, t)(u);
	}
}
function zi(n) {
	n.exit("blockQuote");
}
const Ut = { name: "characterEscape", tokenize: Ri };
function Ri(n, e, t) {
	return r;
	function r(l) {
		return (
			n.enter("characterEscape"),
			n.enter("escapeMarker"),
			n.consume(l),
			n.exit("escapeMarker"),
			i
		);
	}
	function i(l) {
		return bi(l)
			? (n.enter("characterEscapeValue"),
				n.consume(l),
				n.exit("characterEscapeValue"),
				n.exit("characterEscape"),
				e)
			: t(l);
	}
}
const ut = document.createElement("i");
function Pe(n) {
	const e = "&" + n + ";";
	ut.innerHTML = e;
	const t = ut.textContent;
	return (t.charCodeAt(t.length - 1) === 59 && n !== "semi") || t === e
		? !1
		: t;
}
const qt = { name: "characterReference", tokenize: vi };
function vi(n, e, t) {
	const r = this;
	let i = 0,
		l,
		u;
	return o;
	function o(c) {
		return (
			n.enter("characterReference"),
			n.enter("characterReferenceMarker"),
			n.consume(c),
			n.exit("characterReferenceMarker"),
			a
		);
	}
	function a(c) {
		return c === 35
			? (n.enter("characterReferenceMarkerNumeric"),
				n.consume(c),
				n.exit("characterReferenceMarkerNumeric"),
				s)
			: (n.enter("characterReferenceValue"), (l = 31), (u = Z), f(c));
	}
	function s(c) {
		return c === 88 || c === 120
			? (n.enter("characterReferenceMarkerHexadecimal"),
				n.consume(c),
				n.exit("characterReferenceMarkerHexadecimal"),
				n.enter("characterReferenceValue"),
				(l = 6),
				(u = xi),
				f)
			: (n.enter("characterReferenceValue"), (l = 7), (u = Ce), f(c));
	}
	function f(c) {
		if (c === 59 && i) {
			const p = n.exit("characterReferenceValue");
			return u === Z && !Pe(r.sliceSerialize(p))
				? t(c)
				: (n.enter("characterReferenceMarker"),
					n.consume(c),
					n.exit("characterReferenceMarker"),
					n.exit("characterReference"),
					e);
		}
		return u(c) && i++ < l ? (n.consume(c), f) : t(c);
	}
}
const ot = { tokenize: Mi, partial: !0 },
	at = { name: "codeFenced", tokenize: Bi, concrete: !0 };
function Bi(n, e, t) {
	const r = this,
		i = { tokenize: R, partial: !0 };
	let l = 0,
		u = 0,
		o;
	return a;
	function a(x) {
		return s(x);
	}
	function s(x) {
		const D = r.events[r.events.length - 1];
		return (
			(l =
				D && D[1].type === "linePrefix"
					? D[2].sliceSerialize(D[1], !0).length
					: 0),
			(o = x),
			n.enter("codeFenced"),
			n.enter("codeFencedFence"),
			n.enter("codeFencedFenceSequence"),
			f(x)
		);
	}
	function f(x) {
		return x === o
			? (u++, n.consume(x), f)
			: u < 3
				? t(x)
				: (n.exit("codeFencedFenceSequence"),
					j(x) ? U(n, c, "whitespace")(x) : c(x));
	}
	function c(x) {
		return x === null || z(x)
			? (n.exit("codeFencedFence"), r.interrupt ? e(x) : n.check(ot, y, L)(x))
			: (n.enter("codeFencedFenceInfo"),
				n.enter("chunkString", { contentType: "string" }),
				p(x));
	}
	function p(x) {
		return x === null || z(x)
			? (n.exit("chunkString"), n.exit("codeFencedFenceInfo"), c(x))
			: j(x)
				? (n.exit("chunkString"),
					n.exit("codeFencedFenceInfo"),
					U(n, h, "whitespace")(x))
				: x === 96 && x === o
					? t(x)
					: (n.consume(x), p);
	}
	function h(x) {
		return x === null || z(x)
			? c(x)
			: (n.enter("codeFencedFenceMeta"),
				n.enter("chunkString", { contentType: "string" }),
				d(x));
	}
	function d(x) {
		return x === null || z(x)
			? (n.exit("chunkString"), n.exit("codeFencedFenceMeta"), c(x))
			: x === 96 && x === o
				? t(x)
				: (n.consume(x), d);
	}
	function y(x) {
		return n.attempt(i, L, S)(x);
	}
	function S(x) {
		return n.enter("lineEnding"), n.consume(x), n.exit("lineEnding"), k;
	}
	function k(x) {
		return l > 0 && j(x) ? U(n, A, "linePrefix", l + 1)(x) : A(x);
	}
	function A(x) {
		return x === null || z(x)
			? n.check(ot, y, L)(x)
			: (n.enter("codeFlowValue"), C(x));
	}
	function C(x) {
		return x === null || z(x)
			? (n.exit("codeFlowValue"), A(x))
			: (n.consume(x), C);
	}
	function L(x) {
		return n.exit("codeFenced"), e(x);
	}
	function R(x, D, M) {
		let H = 0;
		return b;
		function b(B) {
			return x.enter("lineEnding"), x.consume(B), x.exit("lineEnding"), I;
		}
		function I(B) {
			return (
				x.enter("codeFencedFence"),
				j(B)
					? U(
							x,
							T,
							"linePrefix",
							r.parser.constructs.disable.null.includes("codeIndented")
								? void 0
								: 4,
						)(B)
					: T(B)
			);
		}
		function T(B) {
			return B === o ? (x.enter("codeFencedFenceSequence"), O(B)) : M(B);
		}
		function O(B) {
			return B === o
				? (H++, x.consume(B), O)
				: H >= u
					? (x.exit("codeFencedFenceSequence"),
						j(B) ? U(x, P, "whitespace")(B) : P(B))
					: M(B);
		}
		function P(B) {
			return B === null || z(B) ? (x.exit("codeFencedFence"), D(B)) : M(B);
		}
	}
}
function Mi(n, e, t) {
	const r = this;
	return i;
	function i(u) {
		return u === null
			? t(u)
			: (n.enter("lineEnding"), n.consume(u), n.exit("lineEnding"), l);
	}
	function l(u) {
		return r.parser.lazy[r.now().line] ? t(u) : e(u);
	}
}
const fe = { name: "codeIndented", tokenize: _i },
	Ni = { tokenize: ji, partial: !0 };
function _i(n, e, t) {
	const r = this;
	return i;
	function i(s) {
		return n.enter("codeIndented"), U(n, l, "linePrefix", 5)(s);
	}
	function l(s) {
		const f = r.events[r.events.length - 1];
		return f &&
			f[1].type === "linePrefix" &&
			f[2].sliceSerialize(f[1], !0).length >= 4
			? u(s)
			: t(s);
	}
	function u(s) {
		return s === null
			? a(s)
			: z(s)
				? n.attempt(Ni, u, a)(s)
				: (n.enter("codeFlowValue"), o(s));
	}
	function o(s) {
		return s === null || z(s)
			? (n.exit("codeFlowValue"), u(s))
			: (n.consume(s), o);
	}
	function a(s) {
		return n.exit("codeIndented"), e(s);
	}
}
function ji(n, e, t) {
	const r = this;
	return i;
	function i(u) {
		return r.parser.lazy[r.now().line]
			? t(u)
			: z(u)
				? (n.enter("lineEnding"), n.consume(u), n.exit("lineEnding"), i)
				: U(n, l, "linePrefix", 5)(u);
	}
	function l(u) {
		const o = r.events[r.events.length - 1];
		return o &&
			o[1].type === "linePrefix" &&
			o[2].sliceSerialize(o[1], !0).length >= 4
			? e(u)
			: z(u)
				? i(u)
				: t(u);
	}
}
const Hi = { name: "codeText", tokenize: Vi, resolve: Ui, previous: qi };
function Ui(n) {
	let e = n.length - 4,
		t = 3,
		r,
		i;
	if (
		(n[t][1].type === "lineEnding" || n[t][1].type === "space") &&
		(n[e][1].type === "lineEnding" || n[e][1].type === "space")
	) {
		for (r = t; ++r < e; )
			if (n[r][1].type === "codeTextData") {
				(n[t][1].type = "codeTextPadding"),
					(n[e][1].type = "codeTextPadding"),
					(t += 2),
					(e -= 2);
				break;
			}
	}
	for (r = t - 1, e++; ++r <= e; )
		i === void 0
			? r !== e && n[r][1].type !== "lineEnding" && (i = r)
			: (r === e || n[r][1].type === "lineEnding") &&
				((n[i][1].type = "codeTextData"),
				r !== i + 2 &&
					((n[i][1].end = n[r - 1][1].end),
					n.splice(i + 2, r - i - 2),
					(e -= r - i - 2),
					(r = i + 2)),
				(i = void 0));
	return n;
}
function qi(n) {
	return (
		n !== 96 ||
		this.events[this.events.length - 1][1].type === "characterEscape"
	);
}
function Vi(n, e, t) {
	let r = 0,
		i,
		l;
	return u;
	function u(c) {
		return n.enter("codeText"), n.enter("codeTextSequence"), o(c);
	}
	function o(c) {
		return c === 96
			? (n.consume(c), r++, o)
			: (n.exit("codeTextSequence"), a(c));
	}
	function a(c) {
		return c === null
			? t(c)
			: c === 32
				? (n.enter("space"), n.consume(c), n.exit("space"), a)
				: c === 96
					? ((l = n.enter("codeTextSequence")), (i = 0), f(c))
					: z(c)
						? (n.enter("lineEnding"), n.consume(c), n.exit("lineEnding"), a)
						: (n.enter("codeTextData"), s(c));
	}
	function s(c) {
		return c === null || c === 32 || c === 96 || z(c)
			? (n.exit("codeTextData"), a(c))
			: (n.consume(c), s);
	}
	function f(c) {
		return c === 96
			? (n.consume(c), i++, f)
			: i === r
				? (n.exit("codeTextSequence"), n.exit("codeText"), e(c))
				: ((l.type = "codeTextData"), s(c));
	}
}
function Vt(n) {
	const e = {};
	let t = -1,
		r,
		i,
		l,
		u,
		o,
		a,
		s;
	for (; ++t < n.length; ) {
		for (; t in e; ) t = e[t];
		if (
			((r = n[t]),
			t &&
				r[1].type === "chunkFlow" &&
				n[t - 1][1].type === "listItemPrefix" &&
				((a = r[1]._tokenizer.events),
				(l = 0),
				l < a.length && a[l][1].type === "lineEndingBlank" && (l += 2),
				l < a.length && a[l][1].type === "content"))
		)
			for (; ++l < a.length && a[l][1].type !== "content"; )
				a[l][1].type === "chunkText" &&
					((a[l][1]._isInFirstContentOfListItem = !0), l++);
		if (r[0] === "enter")
			r[1].contentType && (Object.assign(e, $i(n, t)), (t = e[t]), (s = !0));
		else if (r[1]._container) {
			for (
				l = t, i = void 0;
				l-- &&
				((u = n[l]),
				u[1].type === "lineEnding" || u[1].type === "lineEndingBlank");
			)
				u[0] === "enter" &&
					(i && (n[i][1].type = "lineEndingBlank"),
					(u[1].type = "lineEnding"),
					(i = l));
			i &&
				((r[1].end = Object.assign({}, n[i][1].start)),
				(o = n.slice(i, t)),
				o.unshift(r),
				tn(n, i, t - i + 1, o));
		}
	}
	return !s;
}
function $i(n, e) {
	const t = n[e][1],
		r = n[e][2];
	let i = e - 1;
	const l = [],
		u = t._tokenizer || r.parser[t.contentType](t.start),
		o = u.events,
		a = [],
		s = {};
	let f,
		c,
		p = -1,
		h = t,
		d = 0,
		y = 0;
	const S = [y];
	for (; h; ) {
		for (; n[++i][1] !== h; );
		l.push(i),
			h._tokenizer ||
				((f = r.sliceStream(h)),
				h.next || f.push(null),
				c && u.defineSkip(h.start),
				h._isInFirstContentOfListItem &&
					(u._gfmTasklistFirstContentOfListItem = !0),
				u.write(f),
				h._isInFirstContentOfListItem &&
					(u._gfmTasklistFirstContentOfListItem = void 0)),
			(c = h),
			(h = h.next);
	}
	for (h = t; ++p < o.length; )
		o[p][0] === "exit" &&
			o[p - 1][0] === "enter" &&
			o[p][1].type === o[p - 1][1].type &&
			o[p][1].start.line !== o[p][1].end.line &&
			((y = p + 1),
			S.push(y),
			(h._tokenizer = void 0),
			(h.previous = void 0),
			(h = h.next));
	for (
		u.events = [],
			h ? ((h._tokenizer = void 0), (h.previous = void 0)) : S.pop(),
			p = S.length;
		p--;
	) {
		const k = o.slice(S[p], S[p + 1]),
			A = l.pop();
		a.unshift([A, A + k.length - 1]), tn(n, A, 2, k);
	}
	for (p = -1; ++p < a.length; )
		(s[d + a[p][0]] = d + a[p][1]), (d += a[p][1] - a[p][0] - 1);
	return s;
}
const Wi = { tokenize: Gi, resolve: Xi },
	Qi = { tokenize: Yi, partial: !0 };
function Xi(n) {
	return Vt(n), n;
}
function Gi(n, e) {
	let t;
	return r;
	function r(o) {
		return (
			n.enter("content"),
			(t = n.enter("chunkContent", { contentType: "content" })),
			i(o)
		);
	}
	function i(o) {
		return o === null ? l(o) : z(o) ? n.check(Qi, u, l)(o) : (n.consume(o), i);
	}
	function l(o) {
		return n.exit("chunkContent"), n.exit("content"), e(o);
	}
	function u(o) {
		return (
			n.consume(o),
			n.exit("chunkContent"),
			(t.next = n.enter("chunkContent", {
				contentType: "content",
				previous: t,
			})),
			(t = t.next),
			i
		);
	}
}
function Yi(n, e, t) {
	const r = this;
	return i;
	function i(u) {
		return (
			n.exit("chunkContent"),
			n.enter("lineEnding"),
			n.consume(u),
			n.exit("lineEnding"),
			U(n, l, "linePrefix")
		);
	}
	function l(u) {
		if (u === null || z(u)) return t(u);
		const o = r.events[r.events.length - 1];
		return !r.parser.constructs.disable.null.includes("codeIndented") &&
			o &&
			o[1].type === "linePrefix" &&
			o[2].sliceSerialize(o[1], !0).length >= 4
			? e(u)
			: n.interrupt(r.parser.constructs.flow, t, e)(u);
	}
}
function $t(n, e, t, r, i, l, u, o, a) {
	const s = a || Number.POSITIVE_INFINITY;
	let f = 0;
	return c;
	function c(k) {
		return k === 60
			? (n.enter(r), n.enter(i), n.enter(l), n.consume(k), n.exit(l), p)
			: k === null || k === 32 || k === 41 || Xn(k)
				? t(k)
				: (n.enter(r),
					n.enter(u),
					n.enter(o),
					n.enter("chunkString", { contentType: "string" }),
					y(k));
	}
	function p(k) {
		return k === 62
			? (n.enter(l), n.consume(k), n.exit(l), n.exit(i), n.exit(r), e)
			: (n.enter(o), n.enter("chunkString", { contentType: "string" }), h(k));
	}
	function h(k) {
		return k === 62
			? (n.exit("chunkString"), n.exit(o), p(k))
			: k === null || k === 60 || z(k)
				? t(k)
				: (n.consume(k), k === 92 ? d : h);
	}
	function d(k) {
		return k === 60 || k === 62 || k === 92 ? (n.consume(k), h) : h(k);
	}
	function y(k) {
		return !f && (k === null || k === 41 || $(k))
			? (n.exit("chunkString"), n.exit(o), n.exit(u), n.exit(r), e(k))
			: f < s && k === 40
				? (n.consume(k), f++, y)
				: k === 41
					? (n.consume(k), f--, y)
					: k === null || k === 32 || k === 40 || Xn(k)
						? t(k)
						: (n.consume(k), k === 92 ? S : y);
	}
	function S(k) {
		return k === 40 || k === 41 || k === 92 ? (n.consume(k), y) : y(k);
	}
}
function Wt(n, e, t, r, i, l) {
	const u = this;
	let o = 0,
		a;
	return s;
	function s(h) {
		return n.enter(r), n.enter(i), n.consume(h), n.exit(i), n.enter(l), f;
	}
	function f(h) {
		return o > 999 ||
			h === null ||
			h === 91 ||
			(h === 93 && !a) ||
			(h === 94 && !o && "_hiddenFootnoteSupport" in u.parser.constructs)
			? t(h)
			: h === 93
				? (n.exit(l), n.enter(i), n.consume(h), n.exit(i), n.exit(r), e)
				: z(h)
					? (n.enter("lineEnding"), n.consume(h), n.exit("lineEnding"), f)
					: (n.enter("chunkString", { contentType: "string" }), c(h));
	}
	function c(h) {
		return h === null || h === 91 || h === 93 || z(h) || o++ > 999
			? (n.exit("chunkString"), f(h))
			: (n.consume(h), a || (a = !j(h)), h === 92 ? p : c);
	}
	function p(h) {
		return h === 91 || h === 92 || h === 93 ? (n.consume(h), o++, c) : c(h);
	}
}
function Qt(n, e, t, r, i, l) {
	let u;
	return o;
	function o(p) {
		return p === 34 || p === 39 || p === 40
			? (n.enter(r),
				n.enter(i),
				n.consume(p),
				n.exit(i),
				(u = p === 40 ? 41 : p),
				a)
			: t(p);
	}
	function a(p) {
		return p === u
			? (n.enter(i), n.consume(p), n.exit(i), n.exit(r), e)
			: (n.enter(l), s(p));
	}
	function s(p) {
		return p === u
			? (n.exit(l), a(u))
			: p === null
				? t(p)
				: z(p)
					? (n.enter("lineEnding"),
						n.consume(p),
						n.exit("lineEnding"),
						U(n, s, "linePrefix"))
					: (n.enter("chunkString", { contentType: "string" }), f(p));
	}
	function f(p) {
		return p === u || p === null || z(p)
			? (n.exit("chunkString"), s(p))
			: (n.consume(p), p === 92 ? c : f);
	}
	function c(p) {
		return p === u || p === 92 ? (n.consume(p), f) : f(p);
	}
}
function _n(n, e) {
	let t;
	return r;
	function r(i) {
		return z(i)
			? (n.enter("lineEnding"), n.consume(i), n.exit("lineEnding"), (t = !0), r)
			: j(i)
				? U(n, r, t ? "linePrefix" : "lineSuffix")(i)
				: e(i);
	}
}
function fn(n) {
	return n
		.replace(/[\t\n\r ]+/g, " ")
		.replace(/^ | $/g, "")
		.toLowerCase()
		.toUpperCase();
}
const Ki = { name: "definition", tokenize: Ji },
	Zi = { tokenize: nl, partial: !0 };
function Ji(n, e, t) {
	const r = this;
	let i;
	return l;
	function l(h) {
		return n.enter("definition"), u(h);
	}
	function u(h) {
		return Wt.call(
			r,
			n,
			o,
			t,
			"definitionLabel",
			"definitionLabelMarker",
			"definitionLabelString",
		)(h);
	}
	function o(h) {
		return (
			(i = fn(r.sliceSerialize(r.events[r.events.length - 1][1]).slice(1, -1))),
			h === 58
				? (n.enter("definitionMarker"),
					n.consume(h),
					n.exit("definitionMarker"),
					a)
				: t(h)
		);
	}
	function a(h) {
		return $(h) ? _n(n, s)(h) : s(h);
	}
	function s(h) {
		return $t(
			n,
			f,
			t,
			"definitionDestination",
			"definitionDestinationLiteral",
			"definitionDestinationLiteralMarker",
			"definitionDestinationRaw",
			"definitionDestinationString",
		)(h);
	}
	function f(h) {
		return n.attempt(Zi, c, c)(h);
	}
	function c(h) {
		return j(h) ? U(n, p, "whitespace")(h) : p(h);
	}
	function p(h) {
		return h === null || z(h)
			? (n.exit("definition"), r.parser.defined.push(i), e(h))
			: t(h);
	}
}
function nl(n, e, t) {
	return r;
	function r(o) {
		return $(o) ? _n(n, i)(o) : t(o);
	}
	function i(o) {
		return Qt(
			n,
			l,
			t,
			"definitionTitle",
			"definitionTitleMarker",
			"definitionTitleString",
		)(o);
	}
	function l(o) {
		return j(o) ? U(n, u, "whitespace")(o) : u(o);
	}
	function u(o) {
		return o === null || z(o) ? e(o) : t(o);
	}
}
const el = { name: "hardBreakEscape", tokenize: tl };
function tl(n, e, t) {
	return r;
	function r(l) {
		return n.enter("hardBreakEscape"), n.consume(l), i;
	}
	function i(l) {
		return z(l) ? (n.exit("hardBreakEscape"), e(l)) : t(l);
	}
}
const rl = { name: "headingAtx", tokenize: ll, resolve: il };
function il(n, e) {
	let t = n.length - 2,
		r = 3,
		i,
		l;
	return (
		n[r][1].type === "whitespace" && (r += 2),
		t - 2 > r && n[t][1].type === "whitespace" && (t -= 2),
		n[t][1].type === "atxHeadingSequence" &&
			(r === t - 1 || (t - 4 > r && n[t - 2][1].type === "whitespace")) &&
			(t -= r + 1 === t ? 2 : 4),
		t > r &&
			((i = { type: "atxHeadingText", start: n[r][1].start, end: n[t][1].end }),
			(l = {
				type: "chunkText",
				start: n[r][1].start,
				end: n[t][1].end,
				contentType: "text",
			}),
			tn(n, r, t - r + 1, [
				["enter", i, e],
				["enter", l, e],
				["exit", l, e],
				["exit", i, e],
			])),
		n
	);
}
function ll(n, e, t) {
	let r = 0;
	return i;
	function i(f) {
		return n.enter("atxHeading"), l(f);
	}
	function l(f) {
		return n.enter("atxHeadingSequence"), u(f);
	}
	function u(f) {
		return f === 35 && r++ < 6
			? (n.consume(f), u)
			: f === null || $(f)
				? (n.exit("atxHeadingSequence"), o(f))
				: t(f);
	}
	function o(f) {
		return f === 35
			? (n.enter("atxHeadingSequence"), a(f))
			: f === null || z(f)
				? (n.exit("atxHeading"), e(f))
				: j(f)
					? U(n, o, "whitespace")(f)
					: (n.enter("atxHeadingText"), s(f));
	}
	function a(f) {
		return f === 35 ? (n.consume(f), a) : (n.exit("atxHeadingSequence"), o(f));
	}
	function s(f) {
		return f === null || f === 35 || $(f)
			? (n.exit("atxHeadingText"), o(f))
			: (n.consume(f), s);
	}
}
const ul = [
		"address",
		"article",
		"aside",
		"base",
		"basefont",
		"blockquote",
		"body",
		"caption",
		"center",
		"col",
		"colgroup",
		"dd",
		"details",
		"dialog",
		"dir",
		"div",
		"dl",
		"dt",
		"fieldset",
		"figcaption",
		"figure",
		"footer",
		"form",
		"frame",
		"frameset",
		"h1",
		"h2",
		"h3",
		"h4",
		"h5",
		"h6",
		"head",
		"header",
		"hr",
		"html",
		"iframe",
		"legend",
		"li",
		"link",
		"main",
		"menu",
		"menuitem",
		"nav",
		"noframes",
		"ol",
		"optgroup",
		"option",
		"p",
		"param",
		"search",
		"section",
		"summary",
		"table",
		"tbody",
		"td",
		"tfoot",
		"th",
		"thead",
		"title",
		"tr",
		"track",
		"ul",
	],
	st = ["pre", "script", "style", "textarea"],
	ol = { name: "htmlFlow", tokenize: fl, resolveTo: cl, concrete: !0 },
	al = { tokenize: pl, partial: !0 },
	sl = { tokenize: hl, partial: !0 };
function cl(n) {
	let e = n.length;
	for (; e-- && !(n[e][0] === "enter" && n[e][1].type === "htmlFlow"); );
	return (
		e > 1 &&
			n[e - 2][1].type === "linePrefix" &&
			((n[e][1].start = n[e - 2][1].start),
			(n[e + 1][1].start = n[e - 2][1].start),
			n.splice(e - 2, 2)),
		n
	);
}
function fl(n, e, t) {
	const r = this;
	let i, l, u, o, a;
	return s;
	function s(g) {
		return f(g);
	}
	function f(g) {
		return n.enter("htmlFlow"), n.enter("htmlFlowData"), n.consume(g), c;
	}
	function c(g) {
		return g === 33
			? (n.consume(g), p)
			: g === 47
				? (n.consume(g), (l = !0), y)
				: g === 63
					? (n.consume(g), (i = 3), r.interrupt ? e : m)
					: J(g)
						? (n.consume(g), (u = String.fromCharCode(g)), S)
						: t(g);
	}
	function p(g) {
		return g === 45
			? (n.consume(g), (i = 2), h)
			: g === 91
				? (n.consume(g), (i = 5), (o = 0), d)
				: J(g)
					? (n.consume(g), (i = 4), r.interrupt ? e : m)
					: t(g);
	}
	function h(g) {
		return g === 45 ? (n.consume(g), r.interrupt ? e : m) : t(g);
	}
	function d(g) {
		const sn = "CDATA[";
		return g === sn.charCodeAt(o++)
			? (n.consume(g), o === sn.length ? (r.interrupt ? e : T) : d)
			: t(g);
	}
	function y(g) {
		return J(g) ? (n.consume(g), (u = String.fromCharCode(g)), S) : t(g);
	}
	function S(g) {
		if (g === null || g === 47 || g === 62 || $(g)) {
			const sn = g === 47,
				Dn = u.toLowerCase();
			return !sn && !l && st.includes(Dn)
				? ((i = 1), r.interrupt ? e(g) : T(g))
				: ul.includes(u.toLowerCase())
					? ((i = 6), sn ? (n.consume(g), k) : r.interrupt ? e(g) : T(g))
					: ((i = 7),
						r.interrupt && !r.parser.lazy[r.now().line]
							? t(g)
							: l
								? A(g)
								: C(g));
		}
		return g === 45 || Z(g)
			? (n.consume(g), (u += String.fromCharCode(g)), S)
			: t(g);
	}
	function k(g) {
		return g === 62 ? (n.consume(g), r.interrupt ? e : T) : t(g);
	}
	function A(g) {
		return j(g) ? (n.consume(g), A) : b(g);
	}
	function C(g) {
		return g === 47
			? (n.consume(g), b)
			: g === 58 || g === 95 || J(g)
				? (n.consume(g), L)
				: j(g)
					? (n.consume(g), C)
					: b(g);
	}
	function L(g) {
		return g === 45 || g === 46 || g === 58 || g === 95 || Z(g)
			? (n.consume(g), L)
			: R(g);
	}
	function R(g) {
		return g === 61 ? (n.consume(g), x) : j(g) ? (n.consume(g), R) : C(g);
	}
	function x(g) {
		return g === null || g === 60 || g === 61 || g === 62 || g === 96
			? t(g)
			: g === 34 || g === 39
				? (n.consume(g), (a = g), D)
				: j(g)
					? (n.consume(g), x)
					: M(g);
	}
	function D(g) {
		return g === a
			? (n.consume(g), (a = null), H)
			: g === null || z(g)
				? t(g)
				: (n.consume(g), D);
	}
	function M(g) {
		return g === null ||
			g === 34 ||
			g === 39 ||
			g === 47 ||
			g === 60 ||
			g === 61 ||
			g === 62 ||
			g === 96 ||
			$(g)
			? R(g)
			: (n.consume(g), M);
	}
	function H(g) {
		return g === 47 || g === 62 || j(g) ? C(g) : t(g);
	}
	function b(g) {
		return g === 62 ? (n.consume(g), I) : t(g);
	}
	function I(g) {
		return g === null || z(g) ? T(g) : j(g) ? (n.consume(g), I) : t(g);
	}
	function T(g) {
		return g === 45 && i === 2
			? (n.consume(g), G)
			: g === 60 && i === 1
				? (n.consume(g), Y)
				: g === 62 && i === 4
					? (n.consume(g), an)
					: g === 63 && i === 3
						? (n.consume(g), m)
						: g === 93 && i === 5
							? (n.consume(g), mn)
							: z(g) && (i === 6 || i === 7)
								? (n.exit("htmlFlowData"), n.check(al, gn, O)(g))
								: g === null || z(g)
									? (n.exit("htmlFlowData"), O(g))
									: (n.consume(g), T);
	}
	function O(g) {
		return n.check(sl, P, gn)(g);
	}
	function P(g) {
		return n.enter("lineEnding"), n.consume(g), n.exit("lineEnding"), B;
	}
	function B(g) {
		return g === null || z(g) ? O(g) : (n.enter("htmlFlowData"), T(g));
	}
	function G(g) {
		return g === 45 ? (n.consume(g), m) : T(g);
	}
	function Y(g) {
		return g === 47 ? (n.consume(g), (u = ""), on) : T(g);
	}
	function on(g) {
		if (g === 62) {
			const sn = u.toLowerCase();
			return st.includes(sn) ? (n.consume(g), an) : T(g);
		}
		return J(g) && u.length < 8
			? (n.consume(g), (u += String.fromCharCode(g)), on)
			: T(g);
	}
	function mn(g) {
		return g === 93 ? (n.consume(g), m) : T(g);
	}
	function m(g) {
		return g === 62
			? (n.consume(g), an)
			: g === 45 && i === 2
				? (n.consume(g), m)
				: T(g);
	}
	function an(g) {
		return g === null || z(g)
			? (n.exit("htmlFlowData"), gn(g))
			: (n.consume(g), an);
	}
	function gn(g) {
		return n.exit("htmlFlow"), e(g);
	}
}
function hl(n, e, t) {
	const r = this;
	return i;
	function i(u) {
		return z(u)
			? (n.enter("lineEnding"), n.consume(u), n.exit("lineEnding"), l)
			: t(u);
	}
	function l(u) {
		return r.parser.lazy[r.now().line] ? t(u) : e(u);
	}
}
function pl(n, e, t) {
	return r;
	function r(i) {
		return (
			n.enter("lineEnding"),
			n.consume(i),
			n.exit("lineEnding"),
			n.attempt(Hn, e, t)
		);
	}
}
const ml = { name: "htmlText", tokenize: gl };
function gl(n, e, t) {
	const r = this;
	let i, l, u;
	return o;
	function o(m) {
		return n.enter("htmlText"), n.enter("htmlTextData"), n.consume(m), a;
	}
	function a(m) {
		return m === 33
			? (n.consume(m), s)
			: m === 47
				? (n.consume(m), R)
				: m === 63
					? (n.consume(m), C)
					: J(m)
						? (n.consume(m), M)
						: t(m);
	}
	function s(m) {
		return m === 45
			? (n.consume(m), f)
			: m === 91
				? (n.consume(m), (l = 0), d)
				: J(m)
					? (n.consume(m), A)
					: t(m);
	}
	function f(m) {
		return m === 45 ? (n.consume(m), h) : t(m);
	}
	function c(m) {
		return m === null
			? t(m)
			: m === 45
				? (n.consume(m), p)
				: z(m)
					? ((u = c), Y(m))
					: (n.consume(m), c);
	}
	function p(m) {
		return m === 45 ? (n.consume(m), h) : c(m);
	}
	function h(m) {
		return m === 62 ? G(m) : m === 45 ? p(m) : c(m);
	}
	function d(m) {
		const an = "CDATA[";
		return m === an.charCodeAt(l++)
			? (n.consume(m), l === an.length ? y : d)
			: t(m);
	}
	function y(m) {
		return m === null
			? t(m)
			: m === 93
				? (n.consume(m), S)
				: z(m)
					? ((u = y), Y(m))
					: (n.consume(m), y);
	}
	function S(m) {
		return m === 93 ? (n.consume(m), k) : y(m);
	}
	function k(m) {
		return m === 62 ? G(m) : m === 93 ? (n.consume(m), k) : y(m);
	}
	function A(m) {
		return m === null || m === 62
			? G(m)
			: z(m)
				? ((u = A), Y(m))
				: (n.consume(m), A);
	}
	function C(m) {
		return m === null
			? t(m)
			: m === 63
				? (n.consume(m), L)
				: z(m)
					? ((u = C), Y(m))
					: (n.consume(m), C);
	}
	function L(m) {
		return m === 62 ? G(m) : C(m);
	}
	function R(m) {
		return J(m) ? (n.consume(m), x) : t(m);
	}
	function x(m) {
		return m === 45 || Z(m) ? (n.consume(m), x) : D(m);
	}
	function D(m) {
		return z(m) ? ((u = D), Y(m)) : j(m) ? (n.consume(m), D) : G(m);
	}
	function M(m) {
		return m === 45 || Z(m)
			? (n.consume(m), M)
			: m === 47 || m === 62 || $(m)
				? H(m)
				: t(m);
	}
	function H(m) {
		return m === 47
			? (n.consume(m), G)
			: m === 58 || m === 95 || J(m)
				? (n.consume(m), b)
				: z(m)
					? ((u = H), Y(m))
					: j(m)
						? (n.consume(m), H)
						: G(m);
	}
	function b(m) {
		return m === 45 || m === 46 || m === 58 || m === 95 || Z(m)
			? (n.consume(m), b)
			: I(m);
	}
	function I(m) {
		return m === 61
			? (n.consume(m), T)
			: z(m)
				? ((u = I), Y(m))
				: j(m)
					? (n.consume(m), I)
					: H(m);
	}
	function T(m) {
		return m === null || m === 60 || m === 61 || m === 62 || m === 96
			? t(m)
			: m === 34 || m === 39
				? (n.consume(m), (i = m), O)
				: z(m)
					? ((u = T), Y(m))
					: j(m)
						? (n.consume(m), T)
						: (n.consume(m), P);
	}
	function O(m) {
		return m === i
			? (n.consume(m), (i = void 0), B)
			: m === null
				? t(m)
				: z(m)
					? ((u = O), Y(m))
					: (n.consume(m), O);
	}
	function P(m) {
		return m === null ||
			m === 34 ||
			m === 39 ||
			m === 60 ||
			m === 61 ||
			m === 96
			? t(m)
			: m === 47 || m === 62 || $(m)
				? H(m)
				: (n.consume(m), P);
	}
	function B(m) {
		return m === 47 || m === 62 || $(m) ? H(m) : t(m);
	}
	function G(m) {
		return m === 62
			? (n.consume(m), n.exit("htmlTextData"), n.exit("htmlText"), e)
			: t(m);
	}
	function Y(m) {
		return (
			n.exit("htmlTextData"),
			n.enter("lineEnding"),
			n.consume(m),
			n.exit("lineEnding"),
			on
		);
	}
	function on(m) {
		return j(m)
			? U(
					n,
					mn,
					"linePrefix",
					r.parser.constructs.disable.null.includes("codeIndented")
						? void 0
						: 4,
				)(m)
			: mn(m);
	}
	function mn(m) {
		return n.enter("htmlTextData"), u(m);
	}
}
const Le = { name: "labelEnd", tokenize: wl, resolveTo: bl, resolveAll: xl },
	dl = { tokenize: Sl },
	yl = { tokenize: Cl },
	kl = { tokenize: El };
function xl(n) {
	let e = -1;
	for (; ++e < n.length; ) {
		const t = n[e][1];
		(t.type === "labelImage" ||
			t.type === "labelLink" ||
			t.type === "labelEnd") &&
			(n.splice(e + 1, t.type === "labelImage" ? 4 : 2),
			(t.type = "data"),
			e++);
	}
	return n;
}
function bl(n, e) {
	let t = n.length,
		r = 0,
		i,
		l,
		u,
		o;
	for (; t--; )
		if (((i = n[t][1]), l)) {
			if (i.type === "link" || (i.type === "labelLink" && i._inactive)) break;
			n[t][0] === "enter" && i.type === "labelLink" && (i._inactive = !0);
		} else if (u) {
			if (
				n[t][0] === "enter" &&
				(i.type === "labelImage" || i.type === "labelLink") &&
				!i._balanced &&
				((l = t), i.type !== "labelLink")
			) {
				r = 2;
				break;
			}
		} else i.type === "labelEnd" && (u = t);
	const a = {
			type: n[l][1].type === "labelLink" ? "link" : "image",
			start: Object.assign({}, n[l][1].start),
			end: Object.assign({}, n[n.length - 1][1].end),
		},
		s = {
			type: "label",
			start: Object.assign({}, n[l][1].start),
			end: Object.assign({}, n[u][1].end),
		},
		f = {
			type: "labelText",
			start: Object.assign({}, n[l + r + 2][1].end),
			end: Object.assign({}, n[u - 2][1].start),
		};
	return (
		(o = [
			["enter", a, e],
			["enter", s, e],
		]),
		(o = rn(o, n.slice(l + 1, l + r + 3))),
		(o = rn(o, [["enter", f, e]])),
		(o = rn(
			o,
			Zn(e.parser.constructs.insideSpan.null, n.slice(l + r + 4, u - 3), e),
		)),
		(o = rn(o, [["exit", f, e], n[u - 2], n[u - 1], ["exit", s, e]])),
		(o = rn(o, n.slice(u + 1))),
		(o = rn(o, [["exit", a, e]])),
		tn(n, l, n.length, o),
		n
	);
}
function wl(n, e, t) {
	const r = this;
	let i = r.events.length,
		l,
		u;
	for (; i--; )
		if (
			(r.events[i][1].type === "labelImage" ||
				r.events[i][1].type === "labelLink") &&
			!r.events[i][1]._balanced
		) {
			l = r.events[i][1];
			break;
		}
	return o;
	function o(p) {
		return l
			? l._inactive
				? c(p)
				: ((u = r.parser.defined.includes(
						fn(r.sliceSerialize({ start: l.end, end: r.now() })),
					)),
					n.enter("labelEnd"),
					n.enter("labelMarker"),
					n.consume(p),
					n.exit("labelMarker"),
					n.exit("labelEnd"),
					a)
			: t(p);
	}
	function a(p) {
		return p === 40
			? n.attempt(dl, f, u ? f : c)(p)
			: p === 91
				? n.attempt(yl, f, u ? s : c)(p)
				: u
					? f(p)
					: c(p);
	}
	function s(p) {
		return n.attempt(kl, f, c)(p);
	}
	function f(p) {
		return e(p);
	}
	function c(p) {
		return (l._balanced = !0), t(p);
	}
}
function Sl(n, e, t) {
	return r;
	function r(c) {
		return (
			n.enter("resource"),
			n.enter("resourceMarker"),
			n.consume(c),
			n.exit("resourceMarker"),
			i
		);
	}
	function i(c) {
		return $(c) ? _n(n, l)(c) : l(c);
	}
	function l(c) {
		return c === 41
			? f(c)
			: $t(
					n,
					u,
					o,
					"resourceDestination",
					"resourceDestinationLiteral",
					"resourceDestinationLiteralMarker",
					"resourceDestinationRaw",
					"resourceDestinationString",
					32,
				)(c);
	}
	function u(c) {
		return $(c) ? _n(n, a)(c) : f(c);
	}
	function o(c) {
		return t(c);
	}
	function a(c) {
		return c === 34 || c === 39 || c === 40
			? Qt(
					n,
					s,
					t,
					"resourceTitle",
					"resourceTitleMarker",
					"resourceTitleString",
				)(c)
			: f(c);
	}
	function s(c) {
		return $(c) ? _n(n, f)(c) : f(c);
	}
	function f(c) {
		return c === 41
			? (n.enter("resourceMarker"),
				n.consume(c),
				n.exit("resourceMarker"),
				n.exit("resource"),
				e)
			: t(c);
	}
}
function Cl(n, e, t) {
	const r = this;
	return i;
	function i(o) {
		return Wt.call(
			r,
			n,
			l,
			u,
			"reference",
			"referenceMarker",
			"referenceString",
		)(o);
	}
	function l(o) {
		return r.parser.defined.includes(
			fn(r.sliceSerialize(r.events[r.events.length - 1][1]).slice(1, -1)),
		)
			? e(o)
			: t(o);
	}
	function u(o) {
		return t(o);
	}
}
function El(n, e, t) {
	return r;
	function r(l) {
		return (
			n.enter("reference"),
			n.enter("referenceMarker"),
			n.consume(l),
			n.exit("referenceMarker"),
			i
		);
	}
	function i(l) {
		return l === 93
			? (n.enter("referenceMarker"),
				n.consume(l),
				n.exit("referenceMarker"),
				n.exit("reference"),
				e)
			: t(l);
	}
}
const Al = { name: "labelStartImage", tokenize: Fl, resolveAll: Le.resolveAll };
function Fl(n, e, t) {
	const r = this;
	return i;
	function i(o) {
		return (
			n.enter("labelImage"),
			n.enter("labelImageMarker"),
			n.consume(o),
			n.exit("labelImageMarker"),
			l
		);
	}
	function l(o) {
		return o === 91
			? (n.enter("labelMarker"),
				n.consume(o),
				n.exit("labelMarker"),
				n.exit("labelImage"),
				u)
			: t(o);
	}
	function u(o) {
		return o === 94 && "_hiddenFootnoteSupport" in r.parser.constructs
			? t(o)
			: e(o);
	}
}
const Dl = { name: "labelStartLink", tokenize: Il, resolveAll: Le.resolveAll };
function Il(n, e, t) {
	const r = this;
	return i;
	function i(u) {
		return (
			n.enter("labelLink"),
			n.enter("labelMarker"),
			n.consume(u),
			n.exit("labelMarker"),
			n.exit("labelLink"),
			l
		);
	}
	function l(u) {
		return u === 94 && "_hiddenFootnoteSupport" in r.parser.constructs
			? t(u)
			: e(u);
	}
}
const he = { name: "lineEnding", tokenize: Tl };
function Tl(n, e) {
	return t;
	function t(r) {
		return (
			n.enter("lineEnding"),
			n.consume(r),
			n.exit("lineEnding"),
			U(n, e, "linePrefix")
		);
	}
}
const Qn = { name: "thematicBreak", tokenize: Pl };
function Pl(n, e, t) {
	let r = 0,
		i;
	return l;
	function l(s) {
		return n.enter("thematicBreak"), u(s);
	}
	function u(s) {
		return (i = s), o(s);
	}
	function o(s) {
		return s === i
			? (n.enter("thematicBreakSequence"), a(s))
			: r >= 3 && (s === null || z(s))
				? (n.exit("thematicBreak"), e(s))
				: t(s);
	}
	function a(s) {
		return s === i
			? (n.consume(s), r++, a)
			: (n.exit("thematicBreakSequence"),
				j(s) ? U(n, o, "whitespace")(s) : o(s));
	}
}
const nn = {
		name: "list",
		tokenize: zl,
		continuation: { tokenize: Rl },
		exit: Bl,
	},
	Ll = { tokenize: Ml, partial: !0 },
	Ol = { tokenize: vl, partial: !0 };
function zl(n, e, t) {
	const r = this,
		i = r.events[r.events.length - 1];
	let l =
			i && i[1].type === "linePrefix"
				? i[2].sliceSerialize(i[1], !0).length
				: 0,
		u = 0;
	return o;
	function o(h) {
		const d =
			r.containerState.type ||
			(h === 42 || h === 43 || h === 45 ? "listUnordered" : "listOrdered");
		if (
			d === "listUnordered"
				? !r.containerState.marker || h === r.containerState.marker
				: Ce(h)
		) {
			if (
				(r.containerState.type ||
					((r.containerState.type = d), n.enter(d, { _container: !0 })),
				d === "listUnordered")
			)
				return (
					n.enter("listItemPrefix"),
					h === 42 || h === 45 ? n.check(Qn, t, s)(h) : s(h)
				);
			if (!r.interrupt || h === 49)
				return n.enter("listItemPrefix"), n.enter("listItemValue"), a(h);
		}
		return t(h);
	}
	function a(h) {
		return Ce(h) && ++u < 10
			? (n.consume(h), a)
			: (!r.interrupt || u < 2) &&
					(r.containerState.marker
						? h === r.containerState.marker
						: h === 41 || h === 46)
				? (n.exit("listItemValue"), s(h))
				: t(h);
	}
	function s(h) {
		return (
			n.enter("listItemMarker"),
			n.consume(h),
			n.exit("listItemMarker"),
			(r.containerState.marker = r.containerState.marker || h),
			n.check(Hn, r.interrupt ? t : f, n.attempt(Ll, p, c))
		);
	}
	function f(h) {
		return (r.containerState.initialBlankLine = !0), l++, p(h);
	}
	function c(h) {
		return j(h)
			? (n.enter("listItemPrefixWhitespace"),
				n.consume(h),
				n.exit("listItemPrefixWhitespace"),
				p)
			: t(h);
	}
	function p(h) {
		return (
			(r.containerState.size =
				l + r.sliceSerialize(n.exit("listItemPrefix"), !0).length),
			e(h)
		);
	}
}
function Rl(n, e, t) {
	const r = this;
	return (r.containerState._closeFlow = void 0), n.check(Hn, i, l);
	function i(o) {
		return (
			(r.containerState.furtherBlankLines =
				r.containerState.furtherBlankLines ||
				r.containerState.initialBlankLine),
			U(n, e, "listItemIndent", r.containerState.size + 1)(o)
		);
	}
	function l(o) {
		return r.containerState.furtherBlankLines || !j(o)
			? ((r.containerState.furtherBlankLines = void 0),
				(r.containerState.initialBlankLine = void 0),
				u(o))
			: ((r.containerState.furtherBlankLines = void 0),
				(r.containerState.initialBlankLine = void 0),
				n.attempt(Ol, e, u)(o));
	}
	function u(o) {
		return (
			(r.containerState._closeFlow = !0),
			(r.interrupt = void 0),
			U(
				n,
				n.attempt(nn, e, t),
				"linePrefix",
				r.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4,
			)(o)
		);
	}
}
function vl(n, e, t) {
	const r = this;
	return U(n, i, "listItemIndent", r.containerState.size + 1);
	function i(l) {
		const u = r.events[r.events.length - 1];
		return u &&
			u[1].type === "listItemIndent" &&
			u[2].sliceSerialize(u[1], !0).length === r.containerState.size
			? e(l)
			: t(l);
	}
}
function Bl(n) {
	n.exit(this.containerState.type);
}
function Ml(n, e, t) {
	const r = this;
	return U(
		n,
		i,
		"listItemPrefixWhitespace",
		r.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 5,
	);
	function i(l) {
		const u = r.events[r.events.length - 1];
		return !j(l) && u && u[1].type === "listItemPrefixWhitespace" ? e(l) : t(l);
	}
}
const ct = { name: "setextUnderline", tokenize: _l, resolveTo: Nl };
function Nl(n, e) {
	let t = n.length,
		r,
		i,
		l;
	for (; t--; )
		if (n[t][0] === "enter") {
			if (n[t][1].type === "content") {
				r = t;
				break;
			}
			n[t][1].type === "paragraph" && (i = t);
		} else
			n[t][1].type === "content" && n.splice(t, 1),
				!l && n[t][1].type === "definition" && (l = t);
	const u = {
		type: "setextHeading",
		start: Object.assign({}, n[i][1].start),
		end: Object.assign({}, n[n.length - 1][1].end),
	};
	return (
		(n[i][1].type = "setextHeadingText"),
		l
			? (n.splice(i, 0, ["enter", u, e]),
				n.splice(l + 1, 0, ["exit", n[r][1], e]),
				(n[r][1].end = Object.assign({}, n[l][1].end)))
			: (n[r][1] = u),
		n.push(["exit", u, e]),
		n
	);
}
function _l(n, e, t) {
	const r = this;
	let i;
	return l;
	function l(s) {
		let f = r.events.length,
			c;
		for (; f--; )
			if (
				r.events[f][1].type !== "lineEnding" &&
				r.events[f][1].type !== "linePrefix" &&
				r.events[f][1].type !== "content"
			) {
				c = r.events[f][1].type === "paragraph";
				break;
			}
		return !r.parser.lazy[r.now().line] && (r.interrupt || c)
			? (n.enter("setextHeadingLine"), (i = s), u(s))
			: t(s);
	}
	function u(s) {
		return n.enter("setextHeadingLineSequence"), o(s);
	}
	function o(s) {
		return s === i
			? (n.consume(s), o)
			: (n.exit("setextHeadingLineSequence"),
				j(s) ? U(n, a, "lineSuffix")(s) : a(s));
	}
	function a(s) {
		return s === null || z(s) ? (n.exit("setextHeadingLine"), e(s)) : t(s);
	}
}
const jl = { tokenize: Hl };
function Hl(n) {
	const e = this,
		t = n.attempt(
			Hn,
			r,
			n.attempt(
				this.parser.constructs.flowInitial,
				i,
				U(
					n,
					n.attempt(this.parser.constructs.flow, i, n.attempt(Wi, i)),
					"linePrefix",
				),
			),
		);
	return t;
	function r(l) {
		if (l === null) {
			n.consume(l);
			return;
		}
		return (
			n.enter("lineEndingBlank"),
			n.consume(l),
			n.exit("lineEndingBlank"),
			(e.currentConstruct = void 0),
			t
		);
	}
	function i(l) {
		if (l === null) {
			n.consume(l);
			return;
		}
		return (
			n.enter("lineEnding"),
			n.consume(l),
			n.exit("lineEnding"),
			(e.currentConstruct = void 0),
			t
		);
	}
}
const Ul = { resolveAll: Gt() },
	ql = Xt("string"),
	Vl = Xt("text");
function Xt(n) {
	return { tokenize: e, resolveAll: Gt(n === "text" ? $l : void 0) };
	function e(t) {
		const r = this,
			i = this.parser.constructs[n],
			l = t.attempt(i, u, o);
		return u;
		function u(f) {
			return s(f) ? l(f) : o(f);
		}
		function o(f) {
			if (f === null) {
				t.consume(f);
				return;
			}
			return t.enter("data"), t.consume(f), a;
		}
		function a(f) {
			return s(f) ? (t.exit("data"), l(f)) : (t.consume(f), a);
		}
		function s(f) {
			if (f === null) return !0;
			const c = i[f];
			let p = -1;
			if (c)
				for (; ++p < c.length; ) {
					const h = c[p];
					if (!h.previous || h.previous.call(r, r.previous)) return !0;
				}
			return !1;
		}
	}
}
function Gt(n) {
	return e;
	function e(t, r) {
		let i = -1,
			l;
		for (; ++i <= t.length; )
			l === void 0
				? t[i] && t[i][1].type === "data" && ((l = i), i++)
				: (!t[i] || t[i][1].type !== "data") &&
					(i !== l + 2 &&
						((t[l][1].end = t[i - 1][1].end),
						t.splice(l + 2, i - l - 2),
						(i = l + 2)),
					(l = void 0));
		return n ? n(t, r) : t;
	}
}
function $l(n, e) {
	let t = 0;
	for (; ++t <= n.length; )
		if (
			(t === n.length || n[t][1].type === "lineEnding") &&
			n[t - 1][1].type === "data"
		) {
			const r = n[t - 1][1],
				i = e.sliceStream(r);
			let l = i.length,
				u = -1,
				o = 0,
				a;
			for (; l--; ) {
				const s = i[l];
				if (typeof s == "string") {
					for (u = s.length; s.charCodeAt(u - 1) === 32; ) o++, u--;
					if (u) break;
					u = -1;
				} else if (s === -2) (a = !0), o++;
				else if (s !== -1) {
					l++;
					break;
				}
			}
			if (o) {
				const s = {
					type:
						t === n.length || a || o < 2 ? "lineSuffix" : "hardBreakTrailing",
					start: {
						line: r.end.line,
						column: r.end.column - o,
						offset: r.end.offset - o,
						_index: r.start._index + l,
						_bufferIndex: l ? u : r.start._bufferIndex + u,
					},
					end: Object.assign({}, r.end),
				};
				(r.end = Object.assign({}, s.start)),
					r.start.offset === r.end.offset
						? Object.assign(r, s)
						: (n.splice(t, 0, ["enter", s, e], ["exit", s, e]), (t += 2));
			}
			t++;
		}
	return n;
}
function Wl(n, e, t) {
	let r = Object.assign(
		t ? Object.assign({}, t) : { line: 1, column: 1, offset: 0 },
		{ _index: 0, _bufferIndex: -1 },
	);
	const i = {},
		l = [];
	let u = [],
		o = [];
	const a = {
			consume: A,
			enter: C,
			exit: L,
			attempt: D(R),
			check: D(x),
			interrupt: D(x, { interrupt: !0 }),
		},
		s = {
			previous: null,
			code: null,
			containerState: {},
			events: [],
			parser: n,
			sliceStream: h,
			sliceSerialize: p,
			now: d,
			defineSkip: y,
			write: c,
		};
	let f = e.tokenize.call(s, a);
	return e.resolveAll && l.push(e), s;
	function c(I) {
		return (
			(u = rn(u, I)),
			S(),
			u[u.length - 1] !== null
				? []
				: (M(e, 0), (s.events = Zn(l, s.events, s)), s.events)
		);
	}
	function p(I, T) {
		return Xl(h(I), T);
	}
	function h(I) {
		return Ql(u, I);
	}
	function d() {
		const { line: I, column: T, offset: O, _index: P, _bufferIndex: B } = r;
		return { line: I, column: T, offset: O, _index: P, _bufferIndex: B };
	}
	function y(I) {
		(i[I.line] = I.column), b();
	}
	function S() {
		let I;
		for (; r._index < u.length; ) {
			const T = u[r._index];
			if (typeof T == "string")
				for (
					I = r._index, r._bufferIndex < 0 && (r._bufferIndex = 0);
					r._index === I && r._bufferIndex < T.length;
				)
					k(T.charCodeAt(r._bufferIndex));
			else k(T);
		}
	}
	function k(I) {
		f = f(I);
	}
	function A(I) {
		z(I)
			? (r.line++, (r.column = 1), (r.offset += I === -3 ? 2 : 1), b())
			: I !== -1 && (r.column++, r.offset++),
			r._bufferIndex < 0
				? r._index++
				: (r._bufferIndex++,
					r._bufferIndex === u[r._index].length &&
						((r._bufferIndex = -1), r._index++)),
			(s.previous = I);
	}
	function C(I, T) {
		const O = T || {};
		return (
			(O.type = I),
			(O.start = d()),
			s.events.push(["enter", O, s]),
			o.push(O),
			O
		);
	}
	function L(I) {
		const T = o.pop();
		return (T.end = d()), s.events.push(["exit", T, s]), T;
	}
	function R(I, T) {
		M(I, T.from);
	}
	function x(I, T) {
		T.restore();
	}
	function D(I, T) {
		return O;
		function O(P, B, G) {
			let Y, on, mn, m;
			return Array.isArray(P) ? gn(P) : "tokenize" in P ? gn([P]) : an(P);
			function an(K) {
				return In;
				function In(xn) {
					const Tn = xn !== null && K[xn],
						Pn = xn !== null && K.null,
						ee = [
							...(Array.isArray(Tn) ? Tn : Tn ? [Tn] : []),
							...(Array.isArray(Pn) ? Pn : Pn ? [Pn] : []),
						];
					return gn(ee)(xn);
				}
			}
			function gn(K) {
				return (Y = K), (on = 0), K.length === 0 ? G : g(K[on]);
			}
			function g(K) {
				return In;
				function In(xn) {
					return (
						(m = H()),
						(mn = K),
						K.partial || (s.currentConstruct = K),
						K.name && s.parser.constructs.disable.null.includes(K.name)
							? Dn()
							: K.tokenize.call(
									T ? Object.assign(Object.create(s), T) : s,
									a,
									sn,
									Dn,
								)(xn)
					);
				}
			}
			function sn(K) {
				return I(mn, m), B;
			}
			function Dn(K) {
				return m.restore(), ++on < Y.length ? g(Y[on]) : G;
			}
		}
	}
	function M(I, T) {
		I.resolveAll && !l.includes(I) && l.push(I),
			I.resolve &&
				tn(s.events, T, s.events.length - T, I.resolve(s.events.slice(T), s)),
			I.resolveTo && (s.events = I.resolveTo(s.events, s));
	}
	function H() {
		const I = d(),
			T = s.previous,
			O = s.currentConstruct,
			P = s.events.length,
			B = Array.from(o);
		return { restore: G, from: P };
		function G() {
			(r = I),
				(s.previous = T),
				(s.currentConstruct = O),
				(s.events.length = P),
				(o = B),
				b();
		}
	}
	function b() {
		r.line in i &&
			r.column < 2 &&
			((r.column = i[r.line]), (r.offset += i[r.line] - 1));
	}
}
function Ql(n, e) {
	const t = e.start._index,
		r = e.start._bufferIndex,
		i = e.end._index,
		l = e.end._bufferIndex;
	let u;
	if (t === i) u = [n[t].slice(r, l)];
	else {
		if (((u = n.slice(t, i)), r > -1)) {
			const o = u[0];
			typeof o == "string" ? (u[0] = o.slice(r)) : u.shift();
		}
		l > 0 && u.push(n[i].slice(0, l));
	}
	return u;
}
function Xl(n, e) {
	let t = -1;
	const r = [];
	let i;
	for (; ++t < n.length; ) {
		const l = n[t];
		let u;
		if (typeof l == "string") u = l;
		else
			switch (l) {
				case -5: {
					u = "\r";
					break;
				}
				case -4: {
					u = `
`;
					break;
				}
				case -3: {
					u = `\r
`;
					break;
				}
				case -2: {
					u = e ? " " : "	";
					break;
				}
				case -1: {
					if (!e && i) continue;
					u = " ";
					break;
				}
				default:
					u = String.fromCharCode(l);
			}
		(i = l === -2), r.push(u);
	}
	return r.join("");
}
const Gl = {
		42: nn,
		43: nn,
		45: nn,
		48: nn,
		49: nn,
		50: nn,
		51: nn,
		52: nn,
		53: nn,
		54: nn,
		55: nn,
		56: nn,
		57: nn,
		62: Ht,
	},
	Yl = { 91: Ki },
	Kl = { [-2]: fe, [-1]: fe, 32: fe },
	Zl = {
		35: rl,
		42: Qn,
		45: [ct, Qn],
		60: ol,
		61: ct,
		95: Qn,
		96: at,
		126: at,
	},
	Jl = { 38: qt, 92: Ut },
	nu = {
		[-5]: he,
		[-4]: he,
		[-3]: he,
		33: Al,
		38: qt,
		42: Ee,
		60: [Ii, ml],
		91: Dl,
		92: [el, Ut],
		93: Le,
		95: Ee,
		96: Hi,
	},
	eu = { null: [Ee, Ul] },
	tu = { null: [42, 95] },
	ru = { null: [] },
	iu = Object.freeze(
		Object.defineProperty(
			{
				__proto__: null,
				attentionMarkers: tu,
				contentInitial: Yl,
				disable: ru,
				document: Gl,
				flow: Zl,
				flowInitial: Kl,
				insideSpan: eu,
				string: Jl,
				text: nu,
			},
			Symbol.toStringTag,
			{ value: "Module" },
		),
	);
function lu(n) {
	const t = jt([iu, ...((n || {}).extensions || [])]),
		r = {
			defined: [],
			lazy: {},
			constructs: t,
			content: i(wi),
			document: i(Ci),
			flow: i(jl),
			string: i(ql),
			text: i(Vl),
		};
	return r;
	function i(l) {
		return u;
		function u(o) {
			return Wl(r, l, o);
		}
	}
}
const ft = /[\0\t\n\r]/g;
function uu() {
	let n = 1,
		e = "",
		t = !0,
		r;
	return i;
	function i(l, u, o) {
		const a = [];
		let s, f, c, p, h;
		for (
			l = e + l.toString(u),
				c = 0,
				e = "",
				t && (l.charCodeAt(0) === 65279 && c++, (t = void 0));
			c < l.length;
		) {
			if (
				((ft.lastIndex = c),
				(s = ft.exec(l)),
				(p = s && s.index !== void 0 ? s.index : l.length),
				(h = l.charCodeAt(p)),
				!s)
			) {
				e = l.slice(c);
				break;
			}
			if (h === 10 && c === p && r) a.push(-3), (r = void 0);
			else
				switch (
					(r && (a.push(-5), (r = void 0)),
					c < p && (a.push(l.slice(c, p)), (n += p - c)),
					h)
				) {
					case 0: {
						a.push(65533), n++;
						break;
					}
					case 9: {
						for (f = Math.ceil(n / 4) * 4, a.push(-2); n++ < f; ) a.push(-1);
						break;
					}
					case 10: {
						a.push(-4), (n = 1);
						break;
					}
					default:
						(r = !0), (n = 1);
				}
			c = p + 1;
		}
		return o && (r && a.push(-5), e && a.push(e), a.push(null)), a;
	}
}
function ou(n) {
	for (; !Vt(n); );
	return n;
}
function Yt(n, e) {
	const t = Number.parseInt(n, e);
	return t < 9 ||
		t === 11 ||
		(t > 13 && t < 32) ||
		(t > 126 && t < 160) ||
		(t > 55295 && t < 57344) ||
		(t > 64975 && t < 65008) ||
		(t & 65535) === 65535 ||
		(t & 65535) === 65534 ||
		t > 1114111
		? "�"
		: String.fromCharCode(t);
}
const au = /\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;
function Kt(n) {
	return n.replace(au, su);
}
function su(n, e, t) {
	if (e) return e;
	if (t.charCodeAt(0) === 35) {
		const i = t.charCodeAt(1),
			l = i === 120 || i === 88;
		return Yt(t.slice(l ? 2 : 1), l ? 16 : 10);
	}
	return Pe(t) || n;
}
const Zt = {}.hasOwnProperty,
	cu = (n, e, t) => (
		typeof e != "string" && ((t = e), (e = void 0)),
		fu(t)(
			ou(
				lu(t)
					.document()
					.write(uu()(n, e, !0)),
			),
		)
	);
function fu(n) {
	const e = {
		transforms: [],
		canContainEols: ["emphasis", "fragment", "heading", "paragraph", "strong"],
		enter: {
			autolink: o(qe),
			autolinkProtocol: I,
			autolinkEmail: I,
			atxHeading: o(je),
			blockQuote: o(ee),
			characterEscape: I,
			characterReference: I,
			codeFenced: o(_e),
			codeFencedFenceInfo: a,
			codeFencedFenceMeta: a,
			codeIndented: o(_e, a),
			codeText: o(zr, a),
			codeTextData: I,
			data: I,
			codeFlowValue: I,
			definition: o(Rr),
			definitionDestinationString: a,
			definitionLabelString: a,
			definitionTitleString: a,
			emphasis: o(vr),
			hardBreakEscape: o(He),
			hardBreakTrailing: o(He),
			htmlFlow: o(Ue, a),
			htmlFlowData: I,
			htmlText: o(Ue, a),
			htmlTextData: I,
			image: o(Br),
			label: a,
			link: o(qe),
			listItem: o(Mr),
			listItemValue: d,
			listOrdered: o(Ve, h),
			listUnordered: o(Ve),
			paragraph: o(Nr),
			reference: Dn,
			referenceString: a,
			resourceDestinationString: a,
			resourceTitleString: a,
			setextHeading: o(je),
			strong: o(_r),
			thematicBreak: o(Hr),
		},
		exit: {
			atxHeading: f(),
			atxHeadingSequence: D,
			autolink: f(),
			autolinkEmail: Pn,
			autolinkProtocol: Tn,
			blockQuote: f(),
			characterEscapeValue: T,
			characterReferenceMarkerHexadecimal: In,
			characterReferenceMarkerNumeric: In,
			characterReferenceValue: xn,
			codeFenced: f(A),
			codeFencedFence: k,
			codeFencedFenceInfo: y,
			codeFencedFenceMeta: S,
			codeFlowValue: T,
			codeIndented: f(C),
			codeText: f(Y),
			codeTextData: T,
			data: T,
			definition: f(),
			definitionDestinationString: x,
			definitionLabelString: L,
			definitionTitleString: R,
			emphasis: f(),
			hardBreakEscape: f(P),
			hardBreakTrailing: f(P),
			htmlFlow: f(B),
			htmlFlowData: T,
			htmlText: f(G),
			htmlTextData: T,
			image: f(mn),
			label: an,
			labelText: m,
			lineEnding: O,
			link: f(on),
			listItem: f(),
			listOrdered: f(),
			listUnordered: f(),
			paragraph: f(),
			referenceString: K,
			resourceDestinationString: gn,
			resourceTitleString: g,
			resource: sn,
			setextHeading: f(b),
			setextHeadingLineSequence: H,
			setextHeadingText: M,
			strong: f(),
			thematicBreak: f(),
		},
	};
	Jt(e, (n || {}).mdastExtensions || []);
	const t = {};
	return r;
	function r(w) {
		let F = { type: "root", children: [] };
		const v = {
				stack: [F],
				tokenStack: [],
				config: e,
				enter: s,
				exit: c,
				buffer: a,
				resume: p,
				setData: l,
				getData: u,
			},
			q = [];
		let V = -1;
		for (; ++V < w.length; )
			if (w[V][1].type === "listOrdered" || w[V][1].type === "listUnordered")
				if (w[V][0] === "enter") q.push(V);
				else {
					const cn = q.pop();
					V = i(w, cn, V);
				}
		for (V = -1; ++V < w.length; ) {
			const cn = e[w[V][0]];
			Zt.call(cn, w[V][1].type) &&
				cn[w[V][1].type].call(
					Object.assign({ sliceSerialize: w[V][2].sliceSerialize }, v),
					w[V][1],
				);
		}
		if (v.tokenStack.length > 0) {
			const cn = v.tokenStack[v.tokenStack.length - 1];
			(cn[1] || ht).call(v, void 0, cn[0]);
		}
		for (
			F.position = {
				start: wn(
					w.length > 0 ? w[0][1].start : { line: 1, column: 1, offset: 0 },
				),
				end: wn(
					w.length > 0
						? w[w.length - 2][1].end
						: { line: 1, column: 1, offset: 0 },
				),
			},
				V = -1;
			++V < e.transforms.length;
		)
			F = e.transforms[V](F) || F;
		return F;
	}
	function i(w, F, v) {
		let q = F - 1,
			V = -1,
			cn = !1,
			bn,
			dn,
			vn,
			Bn;
		for (; ++q <= v; ) {
			const Q = w[q];
			if (
				(Q[1].type === "listUnordered" ||
				Q[1].type === "listOrdered" ||
				Q[1].type === "blockQuote"
					? (Q[0] === "enter" ? V++ : V--, (Bn = void 0))
					: Q[1].type === "lineEndingBlank"
						? Q[0] === "enter" &&
							(bn && !Bn && !V && !vn && (vn = q), (Bn = void 0))
						: Q[1].type === "linePrefix" ||
							Q[1].type === "listItemValue" ||
							Q[1].type === "listItemMarker" ||
							Q[1].type === "listItemPrefix" ||
							Q[1].type === "listItemPrefixWhitespace" ||
							(Bn = void 0),
				(!V && Q[0] === "enter" && Q[1].type === "listItemPrefix") ||
					(V === -1 &&
						Q[0] === "exit" &&
						(Q[1].type === "listUnordered" || Q[1].type === "listOrdered")))
			) {
				if (bn) {
					let te = q;
					for (dn = void 0; te--; ) {
						const yn = w[te];
						if (
							yn[1].type === "lineEnding" ||
							yn[1].type === "lineEndingBlank"
						) {
							if (yn[0] === "exit") continue;
							dn && ((w[dn][1].type = "lineEndingBlank"), (cn = !0)),
								(yn[1].type = "lineEnding"),
								(dn = te);
						} else if (
							!(
								yn[1].type === "linePrefix" ||
								yn[1].type === "blockQuotePrefix" ||
								yn[1].type === "blockQuotePrefixWhitespace" ||
								yn[1].type === "blockQuoteMarker" ||
								yn[1].type === "listItemIndent"
							)
						)
							break;
					}
					vn && (!dn || vn < dn) && (bn._spread = !0),
						(bn.end = Object.assign({}, dn ? w[dn][1].start : Q[1].end)),
						w.splice(dn || q, 0, ["exit", bn, Q[2]]),
						q++,
						v++;
				}
				Q[1].type === "listItemPrefix" &&
					((bn = {
						type: "listItem",
						_spread: !1,
						start: Object.assign({}, Q[1].start),
						end: void 0,
					}),
					w.splice(q, 0, ["enter", bn, Q[2]]),
					q++,
					v++,
					(vn = void 0),
					(Bn = !0));
			}
		}
		return (w[F][1]._spread = cn), v;
	}
	function l(w, F) {
		t[w] = F;
	}
	function u(w) {
		return t[w];
	}
	function o(w, F) {
		return v;
		function v(q) {
			s.call(this, w(q), q), F && F.call(this, q);
		}
	}
	function a() {
		this.stack.push({ type: "fragment", children: [] });
	}
	function s(w, F, v) {
		return (
			this.stack[this.stack.length - 1].children.push(w),
			this.stack.push(w),
			this.tokenStack.push([F, v]),
			(w.position = { start: wn(F.start) }),
			w
		);
	}
	function f(w) {
		return F;
		function F(v) {
			w && w.call(this, v), c.call(this, v);
		}
	}
	function c(w, F) {
		const v = this.stack.pop(),
			q = this.tokenStack.pop();
		if (q)
			q[0].type !== w.type &&
				(F ? F.call(this, w, q[0]) : (q[1] || ht).call(this, w, q[0]));
		else
			throw new Error(
				"Cannot close `" +
					w.type +
					"` (" +
					Nn({ start: w.start, end: w.end }) +
					"): it’s not open",
			);
		return (v.position.end = wn(w.end)), v;
	}
	function p() {
		return pi(this.stack.pop());
	}
	function h() {
		l("expectingFirstListItemValue", !0);
	}
	function d(w) {
		if (u("expectingFirstListItemValue")) {
			const F = this.stack[this.stack.length - 2];
			(F.start = Number.parseInt(this.sliceSerialize(w), 10)),
				l("expectingFirstListItemValue");
		}
	}
	function y() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.lang = w;
	}
	function S() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.meta = w;
	}
	function k() {
		u("flowCodeInside") || (this.buffer(), l("flowCodeInside", !0));
	}
	function A() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		(F.value = w.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g, "")), l("flowCodeInside");
	}
	function C() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.value = w.replace(/(\r?\n|\r)$/g, "");
	}
	function L(w) {
		const F = this.resume(),
			v = this.stack[this.stack.length - 1];
		(v.label = F), (v.identifier = fn(this.sliceSerialize(w)).toLowerCase());
	}
	function R() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.title = w;
	}
	function x() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.url = w;
	}
	function D(w) {
		const F = this.stack[this.stack.length - 1];
		if (!F.depth) {
			const v = this.sliceSerialize(w).length;
			F.depth = v;
		}
	}
	function M() {
		l("setextHeadingSlurpLineEnding", !0);
	}
	function H(w) {
		const F = this.stack[this.stack.length - 1];
		F.depth = this.sliceSerialize(w).charCodeAt(0) === 61 ? 1 : 2;
	}
	function b() {
		l("setextHeadingSlurpLineEnding");
	}
	function I(w) {
		const F = this.stack[this.stack.length - 1];
		let v = F.children[F.children.length - 1];
		(!v || v.type !== "text") &&
			((v = jr()), (v.position = { start: wn(w.start) }), F.children.push(v)),
			this.stack.push(v);
	}
	function T(w) {
		const F = this.stack.pop();
		(F.value += this.sliceSerialize(w)), (F.position.end = wn(w.end));
	}
	function O(w) {
		const F = this.stack[this.stack.length - 1];
		if (u("atHardBreak")) {
			const v = F.children[F.children.length - 1];
			(v.position.end = wn(w.end)), l("atHardBreak");
			return;
		}
		!u("setextHeadingSlurpLineEnding") &&
			e.canContainEols.includes(F.type) &&
			(I.call(this, w), T.call(this, w));
	}
	function P() {
		l("atHardBreak", !0);
	}
	function B() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.value = w;
	}
	function G() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.value = w;
	}
	function Y() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.value = w;
	}
	function on() {
		const w = this.stack[this.stack.length - 1];
		if (u("inReference")) {
			const F = u("referenceType") || "shortcut";
			(w.type += "Reference"),
				(w.referenceType = F),
				delete w.url,
				delete w.title;
		} else delete w.identifier, delete w.label;
		l("referenceType");
	}
	function mn() {
		const w = this.stack[this.stack.length - 1];
		if (u("inReference")) {
			const F = u("referenceType") || "shortcut";
			(w.type += "Reference"),
				(w.referenceType = F),
				delete w.url,
				delete w.title;
		} else delete w.identifier, delete w.label;
		l("referenceType");
	}
	function m(w) {
		const F = this.sliceSerialize(w),
			v = this.stack[this.stack.length - 2];
		(v.label = Kt(F)), (v.identifier = fn(F).toLowerCase());
	}
	function an() {
		const w = this.stack[this.stack.length - 1],
			F = this.resume(),
			v = this.stack[this.stack.length - 1];
		if ((l("inReference", !0), v.type === "link")) {
			const q = w.children;
			v.children = q;
		} else v.alt = F;
	}
	function gn() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.url = w;
	}
	function g() {
		const w = this.resume(),
			F = this.stack[this.stack.length - 1];
		F.title = w;
	}
	function sn() {
		l("inReference");
	}
	function Dn() {
		l("referenceType", "collapsed");
	}
	function K(w) {
		const F = this.resume(),
			v = this.stack[this.stack.length - 1];
		(v.label = F),
			(v.identifier = fn(this.sliceSerialize(w)).toLowerCase()),
			l("referenceType", "full");
	}
	function In(w) {
		l("characterReferenceType", w.type);
	}
	function xn(w) {
		const F = this.sliceSerialize(w),
			v = u("characterReferenceType");
		let q;
		v
			? ((q = Yt(F, v === "characterReferenceMarkerNumeric" ? 10 : 16)),
				l("characterReferenceType"))
			: (q = Pe(F));
		const V = this.stack.pop();
		(V.value += q), (V.position.end = wn(w.end));
	}
	function Tn(w) {
		T.call(this, w);
		const F = this.stack[this.stack.length - 1];
		F.url = this.sliceSerialize(w);
	}
	function Pn(w) {
		T.call(this, w);
		const F = this.stack[this.stack.length - 1];
		F.url = "mailto:" + this.sliceSerialize(w);
	}
	function ee() {
		return { type: "blockquote", children: [] };
	}
	function _e() {
		return { type: "code", lang: null, meta: null, value: "" };
	}
	function zr() {
		return { type: "inlineCode", value: "" };
	}
	function Rr() {
		return {
			type: "definition",
			identifier: "",
			label: null,
			title: null,
			url: "",
		};
	}
	function vr() {
		return { type: "emphasis", children: [] };
	}
	function je() {
		return { type: "heading", depth: void 0, children: [] };
	}
	function He() {
		return { type: "break" };
	}
	function Ue() {
		return { type: "html", value: "" };
	}
	function Br() {
		return { type: "image", title: null, url: "", alt: null };
	}
	function qe() {
		return { type: "link", title: null, url: "", children: [] };
	}
	function Ve(w) {
		return {
			type: "list",
			ordered: w.type === "listOrdered",
			start: null,
			spread: w._spread,
			children: [],
		};
	}
	function Mr(w) {
		return { type: "listItem", spread: w._spread, checked: null, children: [] };
	}
	function Nr() {
		return { type: "paragraph", children: [] };
	}
	function _r() {
		return { type: "strong", children: [] };
	}
	function jr() {
		return { type: "text", value: "" };
	}
	function Hr() {
		return { type: "thematicBreak" };
	}
}
function wn(n) {
	return { line: n.line, column: n.column, offset: n.offset };
}
function Jt(n, e) {
	let t = -1;
	for (; ++t < e.length; ) {
		const r = e[t];
		Array.isArray(r) ? Jt(n, r) : hu(n, r);
	}
}
function hu(n, e) {
	let t;
	for (t in e)
		if (Zt.call(e, t)) {
			if (t === "canContainEols") {
				const r = e[t];
				r && n[t].push(...r);
			} else if (t === "transforms") {
				const r = e[t];
				r && n[t].push(...r);
			} else if (t === "enter" || t === "exit") {
				const r = e[t];
				r && Object.assign(n[t], r);
			}
		}
}
function ht(n, e) {
	throw n
		? new Error(
				"Cannot close `" +
					n.type +
					"` (" +
					Nn({ start: n.start, end: n.end }) +
					"): a different token (`" +
					e.type +
					"`, " +
					Nn({ start: e.start, end: e.end }) +
					") is open",
			)
		: new Error(
				"Cannot close document, a token (`" +
					e.type +
					"`, " +
					Nn({ start: e.start, end: e.end }) +
					") is still open",
			);
}
function pu(n) {
	Object.assign(this, {
		Parser: (t) => {
			const r = this.data("settings");
			return cu(
				t,
				Object.assign({}, r, n, {
					extensions: this.data("micromarkExtensions") || [],
					mdastExtensions: this.data("fromMarkdownExtensions") || [],
				}),
			);
		},
	});
}
function mu(n, e) {
	const t = {
		type: "element",
		tagName: "blockquote",
		properties: {},
		children: n.wrap(n.all(e), !0),
	};
	return n.patch(e, t), n.applyData(e, t);
}
function gu(n, e) {
	const t = { type: "element", tagName: "br", properties: {}, children: [] };
	return (
		n.patch(e, t),
		[
			n.applyData(e, t),
			{
				type: "text",
				value: `
`,
			},
		]
	);
}
function du(n, e) {
	const t = e.value
			? e.value +
				`
`
			: "",
		r = e.lang ? e.lang.match(/^[^ \t]+(?=[ \t]|$)/) : null,
		i = {};
	r && (i.className = ["language-" + r]);
	let l = {
		type: "element",
		tagName: "code",
		properties: i,
		children: [{ type: "text", value: t }],
	};
	return (
		e.meta && (l.data = { meta: e.meta }),
		n.patch(e, l),
		(l = n.applyData(e, l)),
		(l = { type: "element", tagName: "pre", properties: {}, children: [l] }),
		n.patch(e, l),
		l
	);
}
function yu(n, e) {
	const t = {
		type: "element",
		tagName: "del",
		properties: {},
		children: n.all(e),
	};
	return n.patch(e, t), n.applyData(e, t);
}
function ku(n, e) {
	const t = {
		type: "element",
		tagName: "em",
		properties: {},
		children: n.all(e),
	};
	return n.patch(e, t), n.applyData(e, t);
}
function zn(n) {
	const e = [];
	let t = -1,
		r = 0,
		i = 0;
	for (; ++t < n.length; ) {
		const l = n.charCodeAt(t);
		let u = "";
		if (l === 37 && Z(n.charCodeAt(t + 1)) && Z(n.charCodeAt(t + 2))) i = 2;
		else if (l < 128)
			/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l)) ||
				(u = String.fromCharCode(l));
		else if (l > 55295 && l < 57344) {
			const o = n.charCodeAt(t + 1);
			l < 56320 && o > 56319 && o < 57344
				? ((u = String.fromCharCode(l, o)), (i = 1))
				: (u = "�");
		} else u = String.fromCharCode(l);
		u &&
			(e.push(n.slice(r, t), encodeURIComponent(u)), (r = t + i + 1), (u = "")),
			i && ((t += i), (i = 0));
	}
	return e.join("") + n.slice(r);
}
function nr(n, e) {
	const t = String(e.identifier).toUpperCase(),
		r = zn(t.toLowerCase()),
		i = n.footnoteOrder.indexOf(t);
	let l;
	i === -1
		? (n.footnoteOrder.push(t),
			(n.footnoteCounts[t] = 1),
			(l = n.footnoteOrder.length))
		: (n.footnoteCounts[t]++, (l = i + 1));
	const u = n.footnoteCounts[t],
		o = {
			type: "element",
			tagName: "a",
			properties: {
				href: "#" + n.clobberPrefix + "fn-" + r,
				id: n.clobberPrefix + "fnref-" + r + (u > 1 ? "-" + u : ""),
				dataFootnoteRef: !0,
				ariaDescribedBy: ["footnote-label"],
			},
			children: [{ type: "text", value: String(l) }],
		};
	n.patch(e, o);
	const a = { type: "element", tagName: "sup", properties: {}, children: [o] };
	return n.patch(e, a), n.applyData(e, a);
}
function xu(n, e) {
	const t = n.footnoteById;
	let r = 1;
	for (; r in t; ) r++;
	const i = String(r);
	return (
		(t[i] = {
			type: "footnoteDefinition",
			identifier: i,
			children: [{ type: "paragraph", children: e.children }],
			position: e.position,
		}),
		nr(n, { type: "footnoteReference", identifier: i, position: e.position })
	);
}
function bu(n, e) {
	const t = {
		type: "element",
		tagName: "h" + e.depth,
		properties: {},
		children: n.all(e),
	};
	return n.patch(e, t), n.applyData(e, t);
}
function wu(n, e) {
	if (n.dangerous) {
		const t = { type: "raw", value: e.value };
		return n.patch(e, t), n.applyData(e, t);
	}
	return null;
}
function er(n, e) {
	const t = e.referenceType;
	let r = "]";
	if (
		(t === "collapsed"
			? (r += "[]")
			: t === "full" && (r += "[" + (e.label || e.identifier) + "]"),
		e.type === "imageReference")
	)
		return { type: "text", value: "![" + e.alt + r };
	const i = n.all(e),
		l = i[0];
	l && l.type === "text"
		? (l.value = "[" + l.value)
		: i.unshift({ type: "text", value: "[" });
	const u = i[i.length - 1];
	return (
		u && u.type === "text"
			? (u.value += r)
			: i.push({ type: "text", value: r }),
		i
	);
}
function Su(n, e) {
	const t = n.definition(e.identifier);
	if (!t) return er(n, e);
	const r = { src: zn(t.url || ""), alt: e.alt };
	t.title !== null && t.title !== void 0 && (r.title = t.title);
	const i = { type: "element", tagName: "img", properties: r, children: [] };
	return n.patch(e, i), n.applyData(e, i);
}
function Cu(n, e) {
	const t = { src: zn(e.url) };
	e.alt !== null && e.alt !== void 0 && (t.alt = e.alt),
		e.title !== null && e.title !== void 0 && (t.title = e.title);
	const r = { type: "element", tagName: "img", properties: t, children: [] };
	return n.patch(e, r), n.applyData(e, r);
}
function Eu(n, e) {
	const t = { type: "text", value: e.value.replace(/\r?\n|\r/g, " ") };
	n.patch(e, t);
	const r = { type: "element", tagName: "code", properties: {}, children: [t] };
	return n.patch(e, r), n.applyData(e, r);
}
function Au(n, e) {
	const t = n.definition(e.identifier);
	if (!t) return er(n, e);
	const r = { href: zn(t.url || "") };
	t.title !== null && t.title !== void 0 && (r.title = t.title);
	const i = {
		type: "element",
		tagName: "a",
		properties: r,
		children: n.all(e),
	};
	return n.patch(e, i), n.applyData(e, i);
}
function Fu(n, e) {
	const t = { href: zn(e.url) };
	e.title !== null && e.title !== void 0 && (t.title = e.title);
	const r = {
		type: "element",
		tagName: "a",
		properties: t,
		children: n.all(e),
	};
	return n.patch(e, r), n.applyData(e, r);
}
function Du(n, e, t) {
	const r = n.all(e),
		i = t ? Iu(t) : tr(e),
		l = {},
		u = [];
	if (typeof e.checked == "boolean") {
		const f = r[0];
		let c;
		f && f.type === "element" && f.tagName === "p"
			? (c = f)
			: ((c = { type: "element", tagName: "p", properties: {}, children: [] }),
				r.unshift(c)),
			c.children.length > 0 && c.children.unshift({ type: "text", value: " " }),
			c.children.unshift({
				type: "element",
				tagName: "input",
				properties: { type: "checkbox", checked: e.checked, disabled: !0 },
				children: [],
			}),
			(l.className = ["task-list-item"]);
	}
	let o = -1;
	for (; ++o < r.length; ) {
		const f = r[o];
		(i || o !== 0 || f.type !== "element" || f.tagName !== "p") &&
			u.push({
				type: "text",
				value: `
`,
			}),
			f.type === "element" && f.tagName === "p" && !i
				? u.push(...f.children)
				: u.push(f);
	}
	const a = r[r.length - 1];
	a &&
		(i || a.type !== "element" || a.tagName !== "p") &&
		u.push({
			type: "text",
			value: `
`,
		});
	const s = { type: "element", tagName: "li", properties: l, children: u };
	return n.patch(e, s), n.applyData(e, s);
}
function Iu(n) {
	let e = !1;
	if (n.type === "list") {
		e = n.spread || !1;
		const t = n.children;
		let r = -1;
		for (; !e && ++r < t.length; ) e = tr(t[r]);
	}
	return e;
}
function tr(n) {
	const e = n.spread;
	return e ?? n.children.length > 1;
}
function Tu(n, e) {
	const t = {},
		r = n.all(e);
	let i = -1;
	for (
		typeof e.start == "number" && e.start !== 1 && (t.start = e.start);
		++i < r.length;
	) {
		const u = r[i];
		if (
			u.type === "element" &&
			u.tagName === "li" &&
			u.properties &&
			Array.isArray(u.properties.className) &&
			u.properties.className.includes("task-list-item")
		) {
			t.className = ["contains-task-list"];
			break;
		}
	}
	const l = {
		type: "element",
		tagName: e.ordered ? "ol" : "ul",
		properties: t,
		children: n.wrap(r, !0),
	};
	return n.patch(e, l), n.applyData(e, l);
}
function Pu(n, e) {
	const t = {
		type: "element",
		tagName: "p",
		properties: {},
		children: n.all(e),
	};
	return n.patch(e, t), n.applyData(e, t);
}
function Lu(n, e) {
	const t = { type: "root", children: n.wrap(n.all(e)) };
	return n.patch(e, t), n.applyData(e, t);
}
function Ou(n, e) {
	const t = {
		type: "element",
		tagName: "strong",
		properties: {},
		children: n.all(e),
	};
	return n.patch(e, t), n.applyData(e, t);
}
const Oe = rr("start"),
	ze = rr("end");
function zu(n) {
	return { start: Oe(n), end: ze(n) };
}
function rr(n) {
	return e;
	function e(t) {
		const r = (t && t.position && t.position[n]) || {};
		return {
			line: r.line || null,
			column: r.column || null,
			offset: r.offset > -1 ? r.offset : null,
		};
	}
}
function Ru(n, e) {
	const t = n.all(e),
		r = t.shift(),
		i = [];
	if (r) {
		const u = {
			type: "element",
			tagName: "thead",
			properties: {},
			children: n.wrap([r], !0),
		};
		n.patch(e.children[0], u), i.push(u);
	}
	if (t.length > 0) {
		const u = {
				type: "element",
				tagName: "tbody",
				properties: {},
				children: n.wrap(t, !0),
			},
			o = Oe(e.children[1]),
			a = ze(e.children[e.children.length - 1]);
		o.line && a.line && (u.position = { start: o, end: a }), i.push(u);
	}
	const l = {
		type: "element",
		tagName: "table",
		properties: {},
		children: n.wrap(i, !0),
	};
	return n.patch(e, l), n.applyData(e, l);
}
function vu(n, e, t) {
	const r = t ? t.children : void 0,
		l = (r ? r.indexOf(e) : 1) === 0 ? "th" : "td",
		u = t && t.type === "table" ? t.align : void 0,
		o = u ? u.length : e.children.length;
	let a = -1;
	const s = [];
	for (; ++a < o; ) {
		const c = e.children[a],
			p = {},
			h = u ? u[a] : void 0;
		h && (p.align = h);
		let d = { type: "element", tagName: l, properties: p, children: [] };
		c && ((d.children = n.all(c)), n.patch(c, d), (d = n.applyData(e, d))),
			s.push(d);
	}
	const f = {
		type: "element",
		tagName: "tr",
		properties: {},
		children: n.wrap(s, !0),
	};
	return n.patch(e, f), n.applyData(e, f);
}
function Bu(n, e) {
	const t = {
		type: "element",
		tagName: "td",
		properties: {},
		children: n.all(e),
	};
	return n.patch(e, t), n.applyData(e, t);
}
const pt = 9,
	mt = 32;
function Mu(n) {
	const e = String(n),
		t = /\r?\n|\r/g;
	let r = t.exec(e),
		i = 0;
	const l = [];
	for (; r; )
		l.push(gt(e.slice(i, r.index), i > 0, !0), r[0]),
			(i = r.index + r[0].length),
			(r = t.exec(e));
	return l.push(gt(e.slice(i), i > 0, !1)), l.join("");
}
function gt(n, e, t) {
	let r = 0,
		i = n.length;
	if (e) {
		let l = n.codePointAt(r);
		for (; l === pt || l === mt; ) r++, (l = n.codePointAt(r));
	}
	if (t) {
		let l = n.codePointAt(i - 1);
		for (; l === pt || l === mt; ) i--, (l = n.codePointAt(i - 1));
	}
	return i > r ? n.slice(r, i) : "";
}
function Nu(n, e) {
	const t = { type: "text", value: Mu(String(e.value)) };
	return n.patch(e, t), n.applyData(e, t);
}
function _u(n, e) {
	const t = { type: "element", tagName: "hr", properties: {}, children: [] };
	return n.patch(e, t), n.applyData(e, t);
}
const ju = {
	blockquote: mu,
	break: gu,
	code: du,
	delete: yu,
	emphasis: ku,
	footnoteReference: nr,
	footnote: xu,
	heading: bu,
	html: wu,
	imageReference: Su,
	image: Cu,
	inlineCode: Eu,
	linkReference: Au,
	link: Fu,
	listItem: Du,
	list: Tu,
	paragraph: Pu,
	root: Lu,
	strong: Ou,
	table: Ru,
	tableCell: Bu,
	tableRow: vu,
	text: Nu,
	thematicBreak: _u,
	toml: qn,
	yaml: qn,
	definition: qn,
	footnoteDefinition: qn,
};
function qn() {
	return null;
}
const Re = (n) => {
	if (n == null) return Vu;
	if (typeof n == "string") return qu(n);
	if (typeof n == "object") return Array.isArray(n) ? Hu(n) : Uu(n);
	if (typeof n == "function") return Jn(n);
	throw new Error("Expected function, string, or object as test");
};
function Hu(n) {
	const e = [];
	let t = -1;
	for (; ++t < n.length; ) e[t] = Re(n[t]);
	return Jn(r);
	function r(...i) {
		let l = -1;
		for (; ++l < e.length; ) if (e[l].call(this, ...i)) return !0;
		return !1;
	}
}
function Uu(n) {
	return Jn(e);
	function e(t) {
		let r;
		for (r in n) if (t[r] !== n[r]) return !1;
		return !0;
	}
}
function qu(n) {
	return Jn(e);
	function e(t) {
		return t && t.type === n;
	}
}
function Jn(n) {
	return e;
	function e(t, ...r) {
		return !!(
			t &&
			typeof t == "object" &&
			"type" in t &&
			n.call(this, t, ...r)
		);
	}
}
function Vu() {
	return !0;
}
const $u = !0,
	dt = !1,
	Wu = "skip",
	ir = (n, e, t, r) => {
		typeof e == "function" &&
			typeof t != "function" &&
			((r = t), (t = e), (e = null));
		const i = Re(e),
			l = r ? -1 : 1;
		u(n, void 0, [])();
		function u(o, a, s) {
			const f = o && typeof o == "object" ? o : {};
			if (typeof f.type == "string") {
				const p =
					typeof f.tagName == "string"
						? f.tagName
						: typeof f.name == "string"
							? f.name
							: void 0;
				Object.defineProperty(c, "name", {
					value: "node (" + (o.type + (p ? "<" + p + ">" : "")) + ")",
				});
			}
			return c;
			function c() {
				let p = [],
					h,
					d,
					y;
				if (
					(!e || i(o, a, s[s.length - 1] || null)) &&
					((p = Qu(t(o, s))), p[0] === dt)
				)
					return p;
				if (o.children && p[0] !== Wu)
					for (
						d = (r ? o.children.length : -1) + l, y = s.concat(o);
						d > -1 && d < o.children.length;
					) {
						if (((h = u(o.children[d], d, y)()), h[0] === dt)) return h;
						d = typeof h[1] == "number" ? h[1] : d + l;
					}
				return p;
			}
		}
	};
function Qu(n) {
	return Array.isArray(n) ? n : typeof n == "number" ? [$u, n] : [n];
}
const ve = (n, e, t, r) => {
	typeof e == "function" &&
		typeof t != "function" &&
		((r = t), (t = e), (e = null)),
		ir(n, e, i, r);
	function i(l, u) {
		const o = u[u.length - 1];
		return t(l, o ? o.children.indexOf(l) : null, o);
	}
};
function Xu(n) {
	return (
		!n ||
		!n.position ||
		!n.position.start ||
		!n.position.start.line ||
		!n.position.start.column ||
		!n.position.end ||
		!n.position.end.line ||
		!n.position.end.column
	);
}
const yt = {}.hasOwnProperty;
function Gu(n) {
	const e = Object.create(null);
	if (!n || !n.type) throw new Error("mdast-util-definitions expected node");
	return (
		ve(n, "definition", (r) => {
			const i = kt(r.identifier);
			i && !yt.call(e, i) && (e[i] = r);
		}),
		t
	);
	function t(r) {
		const i = kt(r);
		return i && yt.call(e, i) ? e[i] : null;
	}
}
function kt(n) {
	return String(n || "").toUpperCase();
}
const Yn = {}.hasOwnProperty;
function Yu(n, e) {
	const t = e || {},
		r = t.allowDangerousHtml || !1,
		i = {};
	return (
		(u.dangerous = r),
		(u.clobberPrefix =
			t.clobberPrefix === void 0 || t.clobberPrefix === null
				? "user-content-"
				: t.clobberPrefix),
		(u.footnoteLabel = t.footnoteLabel || "Footnotes"),
		(u.footnoteLabelTagName = t.footnoteLabelTagName || "h2"),
		(u.footnoteLabelProperties = t.footnoteLabelProperties || {
			className: ["sr-only"],
		}),
		(u.footnoteBackLabel = t.footnoteBackLabel || "Back to content"),
		(u.unknownHandler = t.unknownHandler),
		(u.passThrough = t.passThrough),
		(u.handlers = { ...ju, ...t.handlers }),
		(u.definition = Gu(n)),
		(u.footnoteById = i),
		(u.footnoteOrder = []),
		(u.footnoteCounts = {}),
		(u.patch = Ku),
		(u.applyData = Zu),
		(u.one = o),
		(u.all = a),
		(u.wrap = no),
		(u.augment = l),
		ve(n, "footnoteDefinition", (s) => {
			const f = String(s.identifier).toUpperCase();
			Yn.call(i, f) || (i[f] = s);
		}),
		u
	);
	function l(s, f) {
		if (s && "data" in s && s.data) {
			const c = s.data;
			c.hName &&
				(f.type !== "element" &&
					(f = { type: "element", tagName: "", properties: {}, children: [] }),
				(f.tagName = c.hName)),
				f.type === "element" &&
					c.hProperties &&
					(f.properties = { ...f.properties, ...c.hProperties }),
				"children" in f &&
					f.children &&
					c.hChildren &&
					(f.children = c.hChildren);
		}
		if (s) {
			const c = "type" in s ? s : { position: s };
			Xu(c) || (f.position = { start: Oe(c), end: ze(c) });
		}
		return f;
	}
	function u(s, f, c, p) {
		return (
			Array.isArray(c) && ((p = c), (c = {})),
			l(s, {
				type: "element",
				tagName: f,
				properties: c || {},
				children: p || [],
			})
		);
	}
	function o(s, f) {
		return lr(u, s, f);
	}
	function a(s) {
		return Be(u, s);
	}
}
function Ku(n, e) {
	n.position && (e.position = zu(n));
}
function Zu(n, e) {
	let t = e;
	if (n && n.data) {
		const r = n.data.hName,
			i = n.data.hChildren,
			l = n.data.hProperties;
		typeof r == "string" &&
			(t.type === "element"
				? (t.tagName = r)
				: (t = { type: "element", tagName: r, properties: {}, children: [] })),
			t.type === "element" && l && (t.properties = { ...t.properties, ...l }),
			"children" in t &&
				t.children &&
				i !== null &&
				i !== void 0 &&
				(t.children = i);
	}
	return t;
}
function lr(n, e, t) {
	const r = e && e.type;
	if (!r) throw new Error("Expected node, got `" + e + "`");
	return Yn.call(n.handlers, r)
		? n.handlers[r](n, e, t)
		: n.passThrough && n.passThrough.includes(r)
			? "children" in e
				? { ...e, children: Be(n, e) }
				: e
			: n.unknownHandler
				? n.unknownHandler(n, e, t)
				: Ju(n, e);
}
function Be(n, e) {
	const t = [];
	if ("children" in e) {
		const r = e.children;
		let i = -1;
		for (; ++i < r.length; ) {
			const l = lr(n, r[i], e);
			if (l) {
				if (
					i &&
					r[i - 1].type === "break" &&
					(!Array.isArray(l) &&
						l.type === "text" &&
						(l.value = l.value.replace(/^\s+/, "")),
					!Array.isArray(l) && l.type === "element")
				) {
					const u = l.children[0];
					u && u.type === "text" && (u.value = u.value.replace(/^\s+/, ""));
				}
				Array.isArray(l) ? t.push(...l) : t.push(l);
			}
		}
	}
	return t;
}
function Ju(n, e) {
	const t = e.data || {},
		r =
			"value" in e && !(Yn.call(t, "hProperties") || Yn.call(t, "hChildren"))
				? { type: "text", value: e.value }
				: {
						type: "element",
						tagName: "div",
						properties: {},
						children: Be(n, e),
					};
	return n.patch(e, r), n.applyData(e, r);
}
function no(n, e) {
	const t = [];
	let r = -1;
	for (
		e &&
		t.push({
			type: "text",
			value: `
`,
		});
		++r < n.length;
	)
		r &&
			t.push({
				type: "text",
				value: `
`,
			}),
			t.push(n[r]);
	return (
		e &&
			n.length > 0 &&
			t.push({
				type: "text",
				value: `
`,
			}),
		t
	);
}
function eo(n) {
	const e = [];
	let t = -1;
	for (; ++t < n.footnoteOrder.length; ) {
		const r = n.footnoteById[n.footnoteOrder[t]];
		if (!r) continue;
		const i = n.all(r),
			l = String(r.identifier).toUpperCase(),
			u = zn(l.toLowerCase());
		let o = 0;
		const a = [];
		for (; ++o <= n.footnoteCounts[l]; ) {
			const c = {
				type: "element",
				tagName: "a",
				properties: {
					href: "#" + n.clobberPrefix + "fnref-" + u + (o > 1 ? "-" + o : ""),
					dataFootnoteBackref: !0,
					className: ["data-footnote-backref"],
					ariaLabel: n.footnoteBackLabel,
				},
				children: [{ type: "text", value: "↩" }],
			};
			o > 1 &&
				c.children.push({
					type: "element",
					tagName: "sup",
					children: [{ type: "text", value: String(o) }],
				}),
				a.length > 0 && a.push({ type: "text", value: " " }),
				a.push(c);
		}
		const s = i[i.length - 1];
		if (s && s.type === "element" && s.tagName === "p") {
			const c = s.children[s.children.length - 1];
			c && c.type === "text"
				? (c.value += " ")
				: s.children.push({ type: "text", value: " " }),
				s.children.push(...a);
		} else i.push(...a);
		const f = {
			type: "element",
			tagName: "li",
			properties: { id: n.clobberPrefix + "fn-" + u },
			children: n.wrap(i, !0),
		};
		n.patch(r, f), e.push(f);
	}
	if (e.length !== 0)
		return {
			type: "element",
			tagName: "section",
			properties: { dataFootnotes: !0, className: ["footnotes"] },
			children: [
				{
					type: "element",
					tagName: n.footnoteLabelTagName,
					properties: {
						...JSON.parse(JSON.stringify(n.footnoteLabelProperties)),
						id: "footnote-label",
					},
					children: [{ type: "text", value: n.footnoteLabel }],
				},
				{
					type: "text",
					value: `
`,
				},
				{
					type: "element",
					tagName: "ol",
					properties: {},
					children: n.wrap(e, !0),
				},
				{
					type: "text",
					value: `
`,
				},
			],
		};
}
function ur(n, e) {
	const t = Yu(n, e),
		r = t.one(n, null),
		i = eo(t);
	return (
		i &&
			r.children.push(
				{
					type: "text",
					value: `
`,
				},
				i,
			),
		Array.isArray(r) ? { type: "root", children: r } : r
	);
}
const to = (n, e) => (n && "run" in n ? ro(n, e) : io(n || e));
function ro(n, e) {
	return (t, r, i) => {
		n.run(ur(t, e), r, (l) => {
			i(l);
		});
	};
}
function io(n) {
	return (e) => ur(e, n);
}
class Un {
	constructor(e, t, r) {
		(this.property = e), (this.normal = t), r && (this.space = r);
	}
}
Un.prototype.property = {};
Un.prototype.normal = {};
Un.prototype.space = null;
function or(n, e) {
	const t = {},
		r = {};
	let i = -1;
	for (; ++i < n.length; )
		Object.assign(t, n[i].property), Object.assign(r, n[i].normal);
	return new Un(t, r, e);
}
function Ae(n) {
	return n.toLowerCase();
}
class un {
	constructor(e, t) {
		(this.property = e), (this.attribute = t);
	}
}
un.prototype.space = null;
un.prototype.boolean = !1;
un.prototype.booleanish = !1;
un.prototype.overloadedBoolean = !1;
un.prototype.number = !1;
un.prototype.commaSeparated = !1;
un.prototype.spaceSeparated = !1;
un.prototype.commaOrSpaceSeparated = !1;
un.prototype.mustUseProperty = !1;
un.prototype.defined = !1;
let lo = 0;
const _ = Fn(),
	X = Fn(),
	ar = Fn(),
	E = Fn(),
	W = Fn(),
	On = Fn(),
	en = Fn();
function Fn() {
	return 2 ** ++lo;
}
const Fe = Object.freeze(
		Object.defineProperty(
			{
				__proto__: null,
				boolean: _,
				booleanish: X,
				commaOrSpaceSeparated: en,
				commaSeparated: On,
				number: E,
				overloadedBoolean: ar,
				spaceSeparated: W,
			},
			Symbol.toStringTag,
			{ value: "Module" },
		),
	),
	pe = Object.keys(Fe);
class Me extends un {
	constructor(e, t, r, i) {
		let l = -1;
		if ((super(e, t), xt(this, "space", i), typeof r == "number"))
			for (; ++l < pe.length; ) {
				const u = pe[l];
				xt(this, pe[l], (r & Fe[u]) === Fe[u]);
			}
	}
}
Me.prototype.defined = !0;
function xt(n, e, t) {
	t && (n[e] = t);
}
const uo = {}.hasOwnProperty;
function Rn(n) {
	const e = {},
		t = {};
	let r;
	for (r in n.properties)
		if (uo.call(n.properties, r)) {
			const i = n.properties[r],
				l = new Me(r, n.transform(n.attributes || {}, r), i, n.space);
			n.mustUseProperty &&
				n.mustUseProperty.includes(r) &&
				(l.mustUseProperty = !0),
				(e[r] = l),
				(t[Ae(r)] = r),
				(t[Ae(l.attribute)] = r);
		}
	return new Un(e, t, n.space);
}
const sr = Rn({
		space: "xlink",
		transform(n, e) {
			return "xlink:" + e.slice(5).toLowerCase();
		},
		properties: {
			xLinkActuate: null,
			xLinkArcRole: null,
			xLinkHref: null,
			xLinkRole: null,
			xLinkShow: null,
			xLinkTitle: null,
			xLinkType: null,
		},
	}),
	cr = Rn({
		space: "xml",
		transform(n, e) {
			return "xml:" + e.slice(3).toLowerCase();
		},
		properties: { xmlLang: null, xmlBase: null, xmlSpace: null },
	});
function fr(n, e) {
	return e in n ? n[e] : e;
}
function hr(n, e) {
	return fr(n, e.toLowerCase());
}
const pr = Rn({
		space: "xmlns",
		attributes: { xmlnsxlink: "xmlns:xlink" },
		transform: hr,
		properties: { xmlns: null, xmlnsXLink: null },
	}),
	mr = Rn({
		transform(n, e) {
			return e === "role" ? e : "aria-" + e.slice(4).toLowerCase();
		},
		properties: {
			ariaActiveDescendant: null,
			ariaAtomic: X,
			ariaAutoComplete: null,
			ariaBusy: X,
			ariaChecked: X,
			ariaColCount: E,
			ariaColIndex: E,
			ariaColSpan: E,
			ariaControls: W,
			ariaCurrent: null,
			ariaDescribedBy: W,
			ariaDetails: null,
			ariaDisabled: X,
			ariaDropEffect: W,
			ariaErrorMessage: null,
			ariaExpanded: X,
			ariaFlowTo: W,
			ariaGrabbed: X,
			ariaHasPopup: null,
			ariaHidden: X,
			ariaInvalid: null,
			ariaKeyShortcuts: null,
			ariaLabel: null,
			ariaLabelledBy: W,
			ariaLevel: E,
			ariaLive: null,
			ariaModal: X,
			ariaMultiLine: X,
			ariaMultiSelectable: X,
			ariaOrientation: null,
			ariaOwns: W,
			ariaPlaceholder: null,
			ariaPosInSet: E,
			ariaPressed: X,
			ariaReadOnly: X,
			ariaRelevant: null,
			ariaRequired: X,
			ariaRoleDescription: W,
			ariaRowCount: E,
			ariaRowIndex: E,
			ariaRowSpan: E,
			ariaSelected: X,
			ariaSetSize: E,
			ariaSort: null,
			ariaValueMax: E,
			ariaValueMin: E,
			ariaValueNow: E,
			ariaValueText: null,
			role: null,
		},
	}),
	oo = Rn({
		space: "html",
		attributes: {
			acceptcharset: "accept-charset",
			classname: "class",
			htmlfor: "for",
			httpequiv: "http-equiv",
		},
		transform: hr,
		mustUseProperty: ["checked", "multiple", "muted", "selected"],
		properties: {
			abbr: null,
			accept: On,
			acceptCharset: W,
			accessKey: W,
			action: null,
			allow: null,
			allowFullScreen: _,
			allowPaymentRequest: _,
			allowUserMedia: _,
			alt: null,
			as: null,
			async: _,
			autoCapitalize: null,
			autoComplete: W,
			autoFocus: _,
			autoPlay: _,
			blocking: W,
			capture: _,
			charSet: null,
			checked: _,
			cite: null,
			className: W,
			cols: E,
			colSpan: null,
			content: null,
			contentEditable: X,
			controls: _,
			controlsList: W,
			coords: E | On,
			crossOrigin: null,
			data: null,
			dateTime: null,
			decoding: null,
			default: _,
			defer: _,
			dir: null,
			dirName: null,
			disabled: _,
			download: ar,
			draggable: X,
			encType: null,
			enterKeyHint: null,
			fetchPriority: null,
			form: null,
			formAction: null,
			formEncType: null,
			formMethod: null,
			formNoValidate: _,
			formTarget: null,
			headers: W,
			height: E,
			hidden: _,
			high: E,
			href: null,
			hrefLang: null,
			htmlFor: W,
			httpEquiv: W,
			id: null,
			imageSizes: null,
			imageSrcSet: null,
			inert: _,
			inputMode: null,
			integrity: null,
			is: null,
			isMap: _,
			itemId: null,
			itemProp: W,
			itemRef: W,
			itemScope: _,
			itemType: W,
			kind: null,
			label: null,
			lang: null,
			language: null,
			list: null,
			loading: null,
			loop: _,
			low: E,
			manifest: null,
			max: null,
			maxLength: E,
			media: null,
			method: null,
			min: null,
			minLength: E,
			multiple: _,
			muted: _,
			name: null,
			nonce: null,
			noModule: _,
			noValidate: _,
			onAbort: null,
			onAfterPrint: null,
			onAuxClick: null,
			onBeforeMatch: null,
			onBeforePrint: null,
			onBeforeToggle: null,
			onBeforeUnload: null,
			onBlur: null,
			onCancel: null,
			onCanPlay: null,
			onCanPlayThrough: null,
			onChange: null,
			onClick: null,
			onClose: null,
			onContextLost: null,
			onContextMenu: null,
			onContextRestored: null,
			onCopy: null,
			onCueChange: null,
			onCut: null,
			onDblClick: null,
			onDrag: null,
			onDragEnd: null,
			onDragEnter: null,
			onDragExit: null,
			onDragLeave: null,
			onDragOver: null,
			onDragStart: null,
			onDrop: null,
			onDurationChange: null,
			onEmptied: null,
			onEnded: null,
			onError: null,
			onFocus: null,
			onFormData: null,
			onHashChange: null,
			onInput: null,
			onInvalid: null,
			onKeyDown: null,
			onKeyPress: null,
			onKeyUp: null,
			onLanguageChange: null,
			onLoad: null,
			onLoadedData: null,
			onLoadedMetadata: null,
			onLoadEnd: null,
			onLoadStart: null,
			onMessage: null,
			onMessageError: null,
			onMouseDown: null,
			onMouseEnter: null,
			onMouseLeave: null,
			onMouseMove: null,
			onMouseOut: null,
			onMouseOver: null,
			onMouseUp: null,
			onOffline: null,
			onOnline: null,
			onPageHide: null,
			onPageShow: null,
			onPaste: null,
			onPause: null,
			onPlay: null,
			onPlaying: null,
			onPopState: null,
			onProgress: null,
			onRateChange: null,
			onRejectionHandled: null,
			onReset: null,
			onResize: null,
			onScroll: null,
			onScrollEnd: null,
			onSecurityPolicyViolation: null,
			onSeeked: null,
			onSeeking: null,
			onSelect: null,
			onSlotChange: null,
			onStalled: null,
			onStorage: null,
			onSubmit: null,
			onSuspend: null,
			onTimeUpdate: null,
			onToggle: null,
			onUnhandledRejection: null,
			onUnload: null,
			onVolumeChange: null,
			onWaiting: null,
			onWheel: null,
			open: _,
			optimum: E,
			pattern: null,
			ping: W,
			placeholder: null,
			playsInline: _,
			popover: null,
			popoverTarget: null,
			popoverTargetAction: null,
			poster: null,
			preload: null,
			readOnly: _,
			referrerPolicy: null,
			rel: W,
			required: _,
			reversed: _,
			rows: E,
			rowSpan: E,
			sandbox: W,
			scope: null,
			scoped: _,
			seamless: _,
			selected: _,
			shadowRootDelegatesFocus: _,
			shadowRootMode: null,
			shape: null,
			size: E,
			sizes: null,
			slot: null,
			span: E,
			spellCheck: X,
			src: null,
			srcDoc: null,
			srcLang: null,
			srcSet: null,
			start: E,
			step: null,
			style: null,
			tabIndex: E,
			target: null,
			title: null,
			translate: null,
			type: null,
			typeMustMatch: _,
			useMap: null,
			value: X,
			width: E,
			wrap: null,
			align: null,
			aLink: null,
			archive: W,
			axis: null,
			background: null,
			bgColor: null,
			border: E,
			borderColor: null,
			bottomMargin: E,
			cellPadding: null,
			cellSpacing: null,
			char: null,
			charOff: null,
			classId: null,
			clear: null,
			code: null,
			codeBase: null,
			codeType: null,
			color: null,
			compact: _,
			declare: _,
			event: null,
			face: null,
			frame: null,
			frameBorder: null,
			hSpace: E,
			leftMargin: E,
			link: null,
			longDesc: null,
			lowSrc: null,
			marginHeight: E,
			marginWidth: E,
			noResize: _,
			noHref: _,
			noShade: _,
			noWrap: _,
			object: null,
			profile: null,
			prompt: null,
			rev: null,
			rightMargin: E,
			rules: null,
			scheme: null,
			scrolling: X,
			standby: null,
			summary: null,
			text: null,
			topMargin: E,
			valueType: null,
			version: null,
			vAlign: null,
			vLink: null,
			vSpace: E,
			allowTransparency: null,
			autoCorrect: null,
			autoSave: null,
			disablePictureInPicture: _,
			disableRemotePlayback: _,
			prefix: null,
			property: null,
			results: E,
			security: null,
			unselectable: null,
		},
	}),
	ao = Rn({
		space: "svg",
		attributes: {
			accentHeight: "accent-height",
			alignmentBaseline: "alignment-baseline",
			arabicForm: "arabic-form",
			baselineShift: "baseline-shift",
			capHeight: "cap-height",
			className: "class",
			clipPath: "clip-path",
			clipRule: "clip-rule",
			colorInterpolation: "color-interpolation",
			colorInterpolationFilters: "color-interpolation-filters",
			colorProfile: "color-profile",
			colorRendering: "color-rendering",
			crossOrigin: "crossorigin",
			dataType: "datatype",
			dominantBaseline: "dominant-baseline",
			enableBackground: "enable-background",
			fillOpacity: "fill-opacity",
			fillRule: "fill-rule",
			floodColor: "flood-color",
			floodOpacity: "flood-opacity",
			fontFamily: "font-family",
			fontSize: "font-size",
			fontSizeAdjust: "font-size-adjust",
			fontStretch: "font-stretch",
			fontStyle: "font-style",
			fontVariant: "font-variant",
			fontWeight: "font-weight",
			glyphName: "glyph-name",
			glyphOrientationHorizontal: "glyph-orientation-horizontal",
			glyphOrientationVertical: "glyph-orientation-vertical",
			hrefLang: "hreflang",
			horizAdvX: "horiz-adv-x",
			horizOriginX: "horiz-origin-x",
			horizOriginY: "horiz-origin-y",
			imageRendering: "image-rendering",
			letterSpacing: "letter-spacing",
			lightingColor: "lighting-color",
			markerEnd: "marker-end",
			markerMid: "marker-mid",
			markerStart: "marker-start",
			navDown: "nav-down",
			navDownLeft: "nav-down-left",
			navDownRight: "nav-down-right",
			navLeft: "nav-left",
			navNext: "nav-next",
			navPrev: "nav-prev",
			navRight: "nav-right",
			navUp: "nav-up",
			navUpLeft: "nav-up-left",
			navUpRight: "nav-up-right",
			onAbort: "onabort",
			onActivate: "onactivate",
			onAfterPrint: "onafterprint",
			onBeforePrint: "onbeforeprint",
			onBegin: "onbegin",
			onCancel: "oncancel",
			onCanPlay: "oncanplay",
			onCanPlayThrough: "oncanplaythrough",
			onChange: "onchange",
			onClick: "onclick",
			onClose: "onclose",
			onCopy: "oncopy",
			onCueChange: "oncuechange",
			onCut: "oncut",
			onDblClick: "ondblclick",
			onDrag: "ondrag",
			onDragEnd: "ondragend",
			onDragEnter: "ondragenter",
			onDragExit: "ondragexit",
			onDragLeave: "ondragleave",
			onDragOver: "ondragover",
			onDragStart: "ondragstart",
			onDrop: "ondrop",
			onDurationChange: "ondurationchange",
			onEmptied: "onemptied",
			onEnd: "onend",
			onEnded: "onended",
			onError: "onerror",
			onFocus: "onfocus",
			onFocusIn: "onfocusin",
			onFocusOut: "onfocusout",
			onHashChange: "onhashchange",
			onInput: "oninput",
			onInvalid: "oninvalid",
			onKeyDown: "onkeydown",
			onKeyPress: "onkeypress",
			onKeyUp: "onkeyup",
			onLoad: "onload",
			onLoadedData: "onloadeddata",
			onLoadedMetadata: "onloadedmetadata",
			onLoadStart: "onloadstart",
			onMessage: "onmessage",
			onMouseDown: "onmousedown",
			onMouseEnter: "onmouseenter",
			onMouseLeave: "onmouseleave",
			onMouseMove: "onmousemove",
			onMouseOut: "onmouseout",
			onMouseOver: "onmouseover",
			onMouseUp: "onmouseup",
			onMouseWheel: "onmousewheel",
			onOffline: "onoffline",
			onOnline: "ononline",
			onPageHide: "onpagehide",
			onPageShow: "onpageshow",
			onPaste: "onpaste",
			onPause: "onpause",
			onPlay: "onplay",
			onPlaying: "onplaying",
			onPopState: "onpopstate",
			onProgress: "onprogress",
			onRateChange: "onratechange",
			onRepeat: "onrepeat",
			onReset: "onreset",
			onResize: "onresize",
			onScroll: "onscroll",
			onSeeked: "onseeked",
			onSeeking: "onseeking",
			onSelect: "onselect",
			onShow: "onshow",
			onStalled: "onstalled",
			onStorage: "onstorage",
			onSubmit: "onsubmit",
			onSuspend: "onsuspend",
			onTimeUpdate: "ontimeupdate",
			onToggle: "ontoggle",
			onUnload: "onunload",
			onVolumeChange: "onvolumechange",
			onWaiting: "onwaiting",
			onZoom: "onzoom",
			overlinePosition: "overline-position",
			overlineThickness: "overline-thickness",
			paintOrder: "paint-order",
			panose1: "panose-1",
			pointerEvents: "pointer-events",
			referrerPolicy: "referrerpolicy",
			renderingIntent: "rendering-intent",
			shapeRendering: "shape-rendering",
			stopColor: "stop-color",
			stopOpacity: "stop-opacity",
			strikethroughPosition: "strikethrough-position",
			strikethroughThickness: "strikethrough-thickness",
			strokeDashArray: "stroke-dasharray",
			strokeDashOffset: "stroke-dashoffset",
			strokeLineCap: "stroke-linecap",
			strokeLineJoin: "stroke-linejoin",
			strokeMiterLimit: "stroke-miterlimit",
			strokeOpacity: "stroke-opacity",
			strokeWidth: "stroke-width",
			tabIndex: "tabindex",
			textAnchor: "text-anchor",
			textDecoration: "text-decoration",
			textRendering: "text-rendering",
			transformOrigin: "transform-origin",
			typeOf: "typeof",
			underlinePosition: "underline-position",
			underlineThickness: "underline-thickness",
			unicodeBidi: "unicode-bidi",
			unicodeRange: "unicode-range",
			unitsPerEm: "units-per-em",
			vAlphabetic: "v-alphabetic",
			vHanging: "v-hanging",
			vIdeographic: "v-ideographic",
			vMathematical: "v-mathematical",
			vectorEffect: "vector-effect",
			vertAdvY: "vert-adv-y",
			vertOriginX: "vert-origin-x",
			vertOriginY: "vert-origin-y",
			wordSpacing: "word-spacing",
			writingMode: "writing-mode",
			xHeight: "x-height",
			playbackOrder: "playbackorder",
			timelineBegin: "timelinebegin",
		},
		transform: fr,
		properties: {
			about: en,
			accentHeight: E,
			accumulate: null,
			additive: null,
			alignmentBaseline: null,
			alphabetic: E,
			amplitude: E,
			arabicForm: null,
			ascent: E,
			attributeName: null,
			attributeType: null,
			azimuth: E,
			bandwidth: null,
			baselineShift: null,
			baseFrequency: null,
			baseProfile: null,
			bbox: null,
			begin: null,
			bias: E,
			by: null,
			calcMode: null,
			capHeight: E,
			className: W,
			clip: null,
			clipPath: null,
			clipPathUnits: null,
			clipRule: null,
			color: null,
			colorInterpolation: null,
			colorInterpolationFilters: null,
			colorProfile: null,
			colorRendering: null,
			content: null,
			contentScriptType: null,
			contentStyleType: null,
			crossOrigin: null,
			cursor: null,
			cx: null,
			cy: null,
			d: null,
			dataType: null,
			defaultAction: null,
			descent: E,
			diffuseConstant: E,
			direction: null,
			display: null,
			dur: null,
			divisor: E,
			dominantBaseline: null,
			download: _,
			dx: null,
			dy: null,
			edgeMode: null,
			editable: null,
			elevation: E,
			enableBackground: null,
			end: null,
			event: null,
			exponent: E,
			externalResourcesRequired: null,
			fill: null,
			fillOpacity: E,
			fillRule: null,
			filter: null,
			filterRes: null,
			filterUnits: null,
			floodColor: null,
			floodOpacity: null,
			focusable: null,
			focusHighlight: null,
			fontFamily: null,
			fontSize: null,
			fontSizeAdjust: null,
			fontStretch: null,
			fontStyle: null,
			fontVariant: null,
			fontWeight: null,
			format: null,
			fr: null,
			from: null,
			fx: null,
			fy: null,
			g1: On,
			g2: On,
			glyphName: On,
			glyphOrientationHorizontal: null,
			glyphOrientationVertical: null,
			glyphRef: null,
			gradientTransform: null,
			gradientUnits: null,
			handler: null,
			hanging: E,
			hatchContentUnits: null,
			hatchUnits: null,
			height: null,
			href: null,
			hrefLang: null,
			horizAdvX: E,
			horizOriginX: E,
			horizOriginY: E,
			id: null,
			ideographic: E,
			imageRendering: null,
			initialVisibility: null,
			in: null,
			in2: null,
			intercept: E,
			k: E,
			k1: E,
			k2: E,
			k3: E,
			k4: E,
			kernelMatrix: en,
			kernelUnitLength: null,
			keyPoints: null,
			keySplines: null,
			keyTimes: null,
			kerning: null,
			lang: null,
			lengthAdjust: null,
			letterSpacing: null,
			lightingColor: null,
			limitingConeAngle: E,
			local: null,
			markerEnd: null,
			markerMid: null,
			markerStart: null,
			markerHeight: null,
			markerUnits: null,
			markerWidth: null,
			mask: null,
			maskContentUnits: null,
			maskUnits: null,
			mathematical: null,
			max: null,
			media: null,
			mediaCharacterEncoding: null,
			mediaContentEncodings: null,
			mediaSize: E,
			mediaTime: null,
			method: null,
			min: null,
			mode: null,
			name: null,
			navDown: null,
			navDownLeft: null,
			navDownRight: null,
			navLeft: null,
			navNext: null,
			navPrev: null,
			navRight: null,
			navUp: null,
			navUpLeft: null,
			navUpRight: null,
			numOctaves: null,
			observer: null,
			offset: null,
			onAbort: null,
			onActivate: null,
			onAfterPrint: null,
			onBeforePrint: null,
			onBegin: null,
			onCancel: null,
			onCanPlay: null,
			onCanPlayThrough: null,
			onChange: null,
			onClick: null,
			onClose: null,
			onCopy: null,
			onCueChange: null,
			onCut: null,
			onDblClick: null,
			onDrag: null,
			onDragEnd: null,
			onDragEnter: null,
			onDragExit: null,
			onDragLeave: null,
			onDragOver: null,
			onDragStart: null,
			onDrop: null,
			onDurationChange: null,
			onEmptied: null,
			onEnd: null,
			onEnded: null,
			onError: null,
			onFocus: null,
			onFocusIn: null,
			onFocusOut: null,
			onHashChange: null,
			onInput: null,
			onInvalid: null,
			onKeyDown: null,
			onKeyPress: null,
			onKeyUp: null,
			onLoad: null,
			onLoadedData: null,
			onLoadedMetadata: null,
			onLoadStart: null,
			onMessage: null,
			onMouseDown: null,
			onMouseEnter: null,
			onMouseLeave: null,
			onMouseMove: null,
			onMouseOut: null,
			onMouseOver: null,
			onMouseUp: null,
			onMouseWheel: null,
			onOffline: null,
			onOnline: null,
			onPageHide: null,
			onPageShow: null,
			onPaste: null,
			onPause: null,
			onPlay: null,
			onPlaying: null,
			onPopState: null,
			onProgress: null,
			onRateChange: null,
			onRepeat: null,
			onReset: null,
			onResize: null,
			onScroll: null,
			onSeeked: null,
			onSeeking: null,
			onSelect: null,
			onShow: null,
			onStalled: null,
			onStorage: null,
			onSubmit: null,
			onSuspend: null,
			onTimeUpdate: null,
			onToggle: null,
			onUnload: null,
			onVolumeChange: null,
			onWaiting: null,
			onZoom: null,
			opacity: null,
			operator: null,
			order: null,
			orient: null,
			orientation: null,
			origin: null,
			overflow: null,
			overlay: null,
			overlinePosition: E,
			overlineThickness: E,
			paintOrder: null,
			panose1: null,
			path: null,
			pathLength: E,
			patternContentUnits: null,
			patternTransform: null,
			patternUnits: null,
			phase: null,
			ping: W,
			pitch: null,
			playbackOrder: null,
			pointerEvents: null,
			points: null,
			pointsAtX: E,
			pointsAtY: E,
			pointsAtZ: E,
			preserveAlpha: null,
			preserveAspectRatio: null,
			primitiveUnits: null,
			propagate: null,
			property: en,
			r: null,
			radius: null,
			referrerPolicy: null,
			refX: null,
			refY: null,
			rel: en,
			rev: en,
			renderingIntent: null,
			repeatCount: null,
			repeatDur: null,
			requiredExtensions: en,
			requiredFeatures: en,
			requiredFonts: en,
			requiredFormats: en,
			resource: null,
			restart: null,
			result: null,
			rotate: null,
			rx: null,
			ry: null,
			scale: null,
			seed: null,
			shapeRendering: null,
			side: null,
			slope: null,
			snapshotTime: null,
			specularConstant: E,
			specularExponent: E,
			spreadMethod: null,
			spacing: null,
			startOffset: null,
			stdDeviation: null,
			stemh: null,
			stemv: null,
			stitchTiles: null,
			stopColor: null,
			stopOpacity: null,
			strikethroughPosition: E,
			strikethroughThickness: E,
			string: null,
			stroke: null,
			strokeDashArray: en,
			strokeDashOffset: null,
			strokeLineCap: null,
			strokeLineJoin: null,
			strokeMiterLimit: E,
			strokeOpacity: E,
			strokeWidth: null,
			style: null,
			surfaceScale: E,
			syncBehavior: null,
			syncBehaviorDefault: null,
			syncMaster: null,
			syncTolerance: null,
			syncToleranceDefault: null,
			systemLanguage: en,
			tabIndex: E,
			tableValues: null,
			target: null,
			targetX: E,
			targetY: E,
			textAnchor: null,
			textDecoration: null,
			textRendering: null,
			textLength: null,
			timelineBegin: null,
			title: null,
			transformBehavior: null,
			type: null,
			typeOf: en,
			to: null,
			transform: null,
			transformOrigin: null,
			u1: null,
			u2: null,
			underlinePosition: E,
			underlineThickness: E,
			unicode: null,
			unicodeBidi: null,
			unicodeRange: null,
			unitsPerEm: E,
			values: null,
			vAlphabetic: E,
			vMathematical: E,
			vectorEffect: null,
			vHanging: E,
			vIdeographic: E,
			version: null,
			vertAdvY: E,
			vertOriginX: E,
			vertOriginY: E,
			viewBox: null,
			viewTarget: null,
			visibility: null,
			width: null,
			widths: null,
			wordSpacing: null,
			writingMode: null,
			x: null,
			x1: null,
			x2: null,
			xChannelSelector: null,
			xHeight: E,
			y: null,
			y1: null,
			y2: null,
			yChannelSelector: null,
			z: null,
			zoomAndPan: null,
		},
	}),
	so = /^data[-\w.:]+$/i,
	bt = /-[a-z]/g,
	co = /[A-Z]/g;
function fo(n, e) {
	const t = Ae(e);
	let r = e,
		i = un;
	if (t in n.normal) return n.property[n.normal[t]];
	if (t.length > 4 && t.slice(0, 4) === "data" && so.test(e)) {
		if (e.charAt(4) === "-") {
			const l = e.slice(5).replace(bt, po);
			r = "data" + l.charAt(0).toUpperCase() + l.slice(1);
		} else {
			const l = e.slice(4);
			if (!bt.test(l)) {
				let u = l.replace(co, ho);
				u.charAt(0) !== "-" && (u = "-" + u), (e = "data" + u);
			}
		}
		i = Me;
	}
	return new i(r, e);
}
function ho(n) {
	return "-" + n.toLowerCase();
}
function po(n) {
	return n.charAt(1).toUpperCase();
}
const wt = {
		classId: "classID",
		dataType: "datatype",
		itemId: "itemID",
		strokeDashArray: "strokeDasharray",
		strokeDashOffset: "strokeDashoffset",
		strokeLineCap: "strokeLinecap",
		strokeLineJoin: "strokeLinejoin",
		strokeMiterLimit: "strokeMiterlimit",
		typeOf: "typeof",
		xLinkActuate: "xlinkActuate",
		xLinkArcRole: "xlinkArcrole",
		xLinkHref: "xlinkHref",
		xLinkRole: "xlinkRole",
		xLinkShow: "xlinkShow",
		xLinkTitle: "xlinkTitle",
		xLinkType: "xlinkType",
		xmlnsXLink: "xmlnsXlink",
	},
	mo = or([cr, sr, pr, mr, oo], "html"),
	go = or([cr, sr, pr, mr, ao], "svg");
function yo(n) {
	if (n.allowedElements && n.disallowedElements)
		throw new TypeError(
			"Only one of `allowedElements` and `disallowedElements` should be defined",
		);
	if (n.allowedElements || n.disallowedElements || n.allowElement)
		return (e) => {
			ve(e, "element", (t, r, i) => {
				const l = i;
				let u;
				if (
					(n.allowedElements
						? (u = !n.allowedElements.includes(t.tagName))
						: n.disallowedElements &&
							(u = n.disallowedElements.includes(t.tagName)),
					!u &&
						n.allowElement &&
						typeof r == "number" &&
						(u = !n.allowElement(t, r, l)),
					u && typeof r == "number")
				)
					return (
						n.unwrapDisallowed && t.children
							? l.children.splice(r, 1, ...t.children)
							: l.children.splice(r, 1),
						r
					);
			});
		};
}
function ko(n) {
	const e = n && typeof n == "object" && n.type === "text" ? n.value || "" : n;
	return typeof e == "string" && e.replace(/[ \t\n\f\r]/g, "") === "";
}
function xo(n) {
	return n.join(" ").trim();
}
function bo(n, e) {
	const t = {};
	return (n[n.length - 1] === "" ? [...n, ""] : n)
		.join((t.padRight ? " " : "") + "," + (t.padLeft === !1 ? "" : " "))
		.trim();
}
var Vn = { exports: {} },
	me,
	St;
function wo() {
	if (St) return me;
	St = 1;
	var n = /\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,
		e = /\n/g,
		t = /^\s*/,
		r = /^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,
		i = /^:\s*/,
		l = /^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,
		u = /^[;\s]*/,
		o = /^\s+|\s+$/g,
		a = `
`,
		s = "/",
		f = "*",
		c = "",
		p = "comment",
		h = "declaration";
	me = (y, S) => {
		if (typeof y != "string")
			throw new TypeError("First argument must be a string");
		if (!y) return [];
		S = S || {};
		var k = 1,
			A = 1;
		function C(O) {
			var P = O.match(e);
			P && (k += P.length);
			var B = O.lastIndexOf(a);
			A = ~B ? O.length - B : A + O.length;
		}
		function L() {
			var O = { line: k, column: A };
			return (P) => ((P.position = new R(O)), M(), P);
		}
		function R(O) {
			(this.start = O),
				(this.end = { line: k, column: A }),
				(this.source = S.source);
		}
		R.prototype.content = y;
		function x(O) {
			var P = new Error(S.source + ":" + k + ":" + A + ": " + O);
			if (
				((P.reason = O),
				(P.filename = S.source),
				(P.line = k),
				(P.column = A),
				(P.source = y),
				!S.silent)
			)
				throw P;
		}
		function D(O) {
			var P = O.exec(y);
			if (P) {
				var B = P[0];
				return C(B), (y = y.slice(B.length)), P;
			}
		}
		function M() {
			D(t);
		}
		function H(O) {
			var P;
			for (O = O || []; (P = b()); ) P !== !1 && O.push(P);
			return O;
		}
		function b() {
			var O = L();
			if (!(s != y.charAt(0) || f != y.charAt(1))) {
				for (
					var P = 2;
					c != y.charAt(P) && (f != y.charAt(P) || s != y.charAt(P + 1));
				)
					++P;
				if (((P += 2), c === y.charAt(P - 1)))
					return x("End of comment missing");
				var B = y.slice(2, P - 2);
				return (
					(A += 2), C(B), (y = y.slice(P)), (A += 2), O({ type: p, comment: B })
				);
			}
		}
		function I() {
			var O = L(),
				P = D(r);
			if (P) {
				if ((b(), !D(i))) return x("property missing ':'");
				var B = D(l),
					G = O({
						type: h,
						property: d(P[0].replace(n, c)),
						value: B ? d(B[0].replace(n, c)) : c,
					});
				return D(u), G;
			}
		}
		function T() {
			var O = [];
			H(O);
			for (var P; (P = I()); ) P !== !1 && (O.push(P), H(O));
			return O;
		}
		return M(), T();
	};
	function d(y) {
		return y ? y.replace(o, c) : c;
	}
	return me;
}
var Ct;
function So() {
	if (Ct) return Vn.exports;
	Ct = 1;
	var n = wo();
	function e(t, r) {
		var i = null;
		if (!t || typeof t != "string") return i;
		for (
			var l, u = n(t), o = typeof r == "function", a, s, f = 0, c = u.length;
			f < c;
			f++
		)
			(l = u[f]),
				(a = l.property),
				(s = l.value),
				o ? r(a, s, l) : s && (i || (i = {}), (i[a] = s));
		return i;
	}
	return (Vn.exports = e), (Vn.exports.default = e), Vn.exports;
}
var Co = So();
const Eo = Te(Co),
	De = {}.hasOwnProperty,
	Ao = new Set(["table", "thead", "tbody", "tfoot", "tr"]);
function gr(n, e) {
	const t = [];
	let r = -1,
		i;
	for (; ++r < e.children.length; )
		(i = e.children[r]),
			i.type === "element"
				? t.push(Fo(n, i, r, e))
				: i.type === "text"
					? (e.type !== "element" || !Ao.has(e.tagName) || !ko(i)) &&
						t.push(i.value)
					: i.type === "raw" && !n.options.skipHtml && t.push(i.value);
	return t;
}
function Fo(n, e, t, r) {
	const i = n.options,
		l = i.transformLinkUri === void 0 ? Vr : i.transformLinkUri,
		u = n.schema,
		o = e.tagName,
		a = {};
	let s = u,
		f;
	if (
		(u.space === "html" && o === "svg" && ((s = go), (n.schema = s)),
		e.properties)
	)
		for (f in e.properties)
			De.call(e.properties, f) && Io(a, f, e.properties[f], n);
	(o === "ol" || o === "ul") && n.listDepth++;
	const c = gr(n, e);
	(o === "ol" || o === "ul") && n.listDepth--, (n.schema = u);
	const p = e.position || {
			start: { line: null, column: null, offset: null },
			end: { line: null, column: null, offset: null },
		},
		h = i.components && De.call(i.components, o) ? i.components[o] : o,
		d = typeof h == "string" || h === En.Fragment;
	if (!Ur.isValidElementType(h))
		throw new TypeError(
			`Component for name \`${o}\` not defined or is not renderable`,
		);
	if (
		((a.key = t),
		o === "a" &&
			i.linkTarget &&
			(a.target =
				typeof i.linkTarget == "function"
					? i.linkTarget(
							String(a.href || ""),
							e.children,
							typeof a.title == "string" ? a.title : null,
						)
					: i.linkTarget),
		o === "a" &&
			l &&
			(a.href = l(
				String(a.href || ""),
				e.children,
				typeof a.title == "string" ? a.title : null,
			)),
		!d &&
			o === "code" &&
			r.type === "element" &&
			r.tagName !== "pre" &&
			(a.inline = !0),
		!d &&
			(o === "h1" ||
				o === "h2" ||
				o === "h3" ||
				o === "h4" ||
				o === "h5" ||
				o === "h6") &&
			(a.level = Number.parseInt(o.charAt(1), 10)),
		o === "img" &&
			i.transformImageUri &&
			(a.src = i.transformImageUri(
				String(a.src || ""),
				String(a.alt || ""),
				typeof a.title == "string" ? a.title : null,
			)),
		!d && o === "li" && r.type === "element")
	) {
		const y = Do(e);
		(a.checked = y && y.properties ? !!y.properties.checked : null),
			(a.index = ge(r, e)),
			(a.ordered = r.tagName === "ol");
	}
	return (
		!d &&
			(o === "ol" || o === "ul") &&
			((a.ordered = o === "ol"), (a.depth = n.listDepth)),
		(o === "td" || o === "th") &&
			(a.align &&
				(a.style || (a.style = {}),
				(a.style.textAlign = a.align),
				delete a.align),
			d || (a.isHeader = o === "th")),
		!d &&
			o === "tr" &&
			r.type === "element" &&
			(a.isHeader = r.tagName === "thead"),
		i.sourcePos && (a["data-sourcepos"] = Lo(p)),
		!d && i.rawSourcePos && (a.sourcePosition = e.position),
		!d &&
			i.includeElementIndex &&
			((a.index = ge(r, e)), (a.siblingCount = ge(r))),
		d || (a.node = e),
		c.length > 0 ? En.createElement(h, a, c) : En.createElement(h, a)
	);
}
function Do(n) {
	let e = -1;
	for (; ++e < n.children.length; ) {
		const t = n.children[e];
		if (t.type === "element" && t.tagName === "input") return t;
	}
	return null;
}
function ge(n, e) {
	let t = -1,
		r = 0;
	for (; ++t < n.children.length && n.children[t] !== e; )
		n.children[t].type === "element" && r++;
	return r;
}
function Io(n, e, t, r) {
	const i = fo(r.schema, e);
	let l = t;
	l == null ||
		l !== l ||
		(Array.isArray(l) && (l = i.commaSeparated ? bo(l) : xo(l)),
		i.property === "style" && typeof l == "string" && (l = To(l)),
		i.space && i.property
			? (n[De.call(wt, i.property) ? wt[i.property] : i.property] = l)
			: i.attribute && (n[i.attribute] = l));
}
function To(n) {
	const e = {};
	try {
		Eo(n, t);
	} catch {}
	return e;
	function t(r, i) {
		const l = r.slice(0, 4) === "-ms-" ? `ms-${r.slice(4)}` : r;
		e[l.replace(/-([a-z])/g, Po)] = i;
	}
}
function Po(n, e) {
	return e.toUpperCase();
}
function Lo(n) {
	return [n.start.line, ":", n.start.column, "-", n.end.line, ":", n.end.column]
		.map(String)
		.join("");
}
const Et = {}.hasOwnProperty,
	Oo = "https://github.com/remarkjs/react-markdown/blob/main/changelog.md",
	$n = {
		plugins: { to: "remarkPlugins", id: "change-plugins-to-remarkplugins" },
		renderers: { to: "components", id: "change-renderers-to-components" },
		astPlugins: { id: "remove-buggy-html-in-markdown-parser" },
		allowDangerousHtml: { id: "remove-buggy-html-in-markdown-parser" },
		escapeHtml: { id: "remove-buggy-html-in-markdown-parser" },
		source: { to: "children", id: "change-source-to-children" },
		allowNode: {
			to: "allowElement",
			id: "replace-allownode-allowedtypes-and-disallowedtypes",
		},
		allowedTypes: {
			to: "allowedElements",
			id: "replace-allownode-allowedtypes-and-disallowedtypes",
		},
		disallowedTypes: {
			to: "disallowedElements",
			id: "replace-allownode-allowedtypes-and-disallowedtypes",
		},
		includeNodeIndex: {
			to: "includeElementIndex",
			id: "change-includenodeindex-to-includeelementindex",
		},
	};
function dr(n) {
	for (const l in $n)
		if (Et.call($n, l) && Et.call(n, l)) {
			const u = $n[l];
			console.warn(
				`[react-markdown] Warning: please ${u.to ? `use \`${u.to}\` instead of` : "remove"} \`${l}\` (see <${Oo}#${u.id}> for more info)`,
			),
				delete $n[l];
		}
	const e = ai()
			.use(pu)
			.use(n.remarkPlugins || [])
			.use(to, { ...n.remarkRehypeOptions, allowDangerousHtml: !0 })
			.use(n.rehypePlugins || [])
			.use(yo, n),
		t = new Bt();
	typeof n.children == "string"
		? (t.value = n.children)
		: n.children !== void 0 &&
			n.children !== null &&
			console.warn(
				`[react-markdown] Warning: please pass a string as \`children\` (not: \`${n.children}\`)`,
			);
	const r = e.runSync(e.parse(t), t);
	if (r.type !== "root") throw new TypeError("Expected a `root` node");
	let i = En.createElement(
		En.Fragment,
		{},
		gr({ options: n, schema: mo, listDepth: 0 }, r),
	);
	return (
		n.className && (i = En.createElement("div", { className: n.className }, i)),
		i
	);
}
dr.propTypes = {
	children: N.string,
	className: N.string,
	allowElement: N.func,
	allowedElements: N.arrayOf(N.string),
	disallowedElements: N.arrayOf(N.string),
	unwrapDisallowed: N.bool,
	remarkPlugins: N.arrayOf(
		N.oneOfType([
			N.object,
			N.func,
			N.arrayOf(
				N.oneOfType([N.bool, N.string, N.object, N.func, N.arrayOf(N.any)]),
			),
		]),
	),
	rehypePlugins: N.arrayOf(
		N.oneOfType([
			N.object,
			N.func,
			N.arrayOf(
				N.oneOfType([N.bool, N.string, N.object, N.func, N.arrayOf(N.any)]),
			),
		]),
	),
	sourcePos: N.bool,
	rawSourcePos: N.bool,
	skipHtml: N.bool,
	includeElementIndex: N.bool,
	transformLinkUri: N.oneOfType([N.func, N.bool]),
	linkTarget: N.oneOfType([N.func, N.string]),
	transformImageUri: N.func,
	components: N.object,
};
const zo = { tokenize: _o, partial: !0 },
	yr = { tokenize: jo, partial: !0 },
	kr = { tokenize: Ho, partial: !0 },
	xr = { tokenize: Uo, partial: !0 },
	Ro = { tokenize: qo, partial: !0 },
	br = { tokenize: Mo, previous: Sr },
	wr = { tokenize: No, previous: Cr },
	kn = { tokenize: Bo, previous: Er },
	pn = {},
	vo = { text: pn };
let Cn = 48;
for (; Cn < 123; )
	(pn[Cn] = kn), Cn++, Cn === 58 ? (Cn = 65) : Cn === 91 && (Cn = 97);
pn[43] = kn;
pn[45] = kn;
pn[46] = kn;
pn[95] = kn;
pn[72] = [kn, wr];
pn[104] = [kn, wr];
pn[87] = [kn, br];
pn[119] = [kn, br];
function Bo(n, e, t) {
	const r = this;
	let i, l;
	return u;
	function u(c) {
		return !Ie(c) || !Er.call(r, r.previous) || Ne(r.events)
			? t(c)
			: (n.enter("literalAutolink"), n.enter("literalAutolinkEmail"), o(c));
	}
	function o(c) {
		return Ie(c) ? (n.consume(c), o) : c === 64 ? (n.consume(c), a) : t(c);
	}
	function a(c) {
		return c === 46
			? n.check(Ro, f, s)(c)
			: c === 45 || c === 95 || Z(c)
				? ((l = !0), n.consume(c), a)
				: f(c);
	}
	function s(c) {
		return n.consume(c), (i = !0), a;
	}
	function f(c) {
		return l && i && J(r.previous)
			? (n.exit("literalAutolinkEmail"), n.exit("literalAutolink"), e(c))
			: t(c);
	}
}
function Mo(n, e, t) {
	const r = this;
	return i;
	function i(u) {
		return (u !== 87 && u !== 119) || !Sr.call(r, r.previous) || Ne(r.events)
			? t(u)
			: (n.enter("literalAutolink"),
				n.enter("literalAutolinkWww"),
				n.check(zo, n.attempt(yr, n.attempt(kr, l), t), t)(u));
	}
	function l(u) {
		return n.exit("literalAutolinkWww"), n.exit("literalAutolink"), e(u);
	}
}
function No(n, e, t) {
	const r = this;
	let i = "",
		l = !1;
	return u;
	function u(c) {
		return (c === 72 || c === 104) && Cr.call(r, r.previous) && !Ne(r.events)
			? (n.enter("literalAutolink"),
				n.enter("literalAutolinkHttp"),
				(i += String.fromCodePoint(c)),
				n.consume(c),
				o)
			: t(c);
	}
	function o(c) {
		if (J(c) && i.length < 5)
			return (i += String.fromCodePoint(c)), n.consume(c), o;
		if (c === 58) {
			const p = i.toLowerCase();
			if (p === "http" || p === "https") return n.consume(c), a;
		}
		return t(c);
	}
	function a(c) {
		return c === 47 ? (n.consume(c), l ? s : ((l = !0), a)) : t(c);
	}
	function s(c) {
		return c === null || Xn(c) || $(c) || An(c) || Kn(c)
			? t(c)
			: n.attempt(yr, n.attempt(kr, f), t)(c);
	}
	function f(c) {
		return n.exit("literalAutolinkHttp"), n.exit("literalAutolink"), e(c);
	}
}
function _o(n, e, t) {
	let r = 0;
	return i;
	function i(u) {
		return (u === 87 || u === 119) && r < 3
			? (r++, n.consume(u), i)
			: u === 46 && r === 3
				? (n.consume(u), l)
				: t(u);
	}
	function l(u) {
		return u === null ? t(u) : e(u);
	}
}
function jo(n, e, t) {
	let r, i, l;
	return u;
	function u(s) {
		return s === 46 || s === 95
			? n.check(xr, a, o)(s)
			: s === null || $(s) || An(s) || (s !== 45 && Kn(s))
				? a(s)
				: ((l = !0), n.consume(s), u);
	}
	function o(s) {
		return s === 95 ? (r = !0) : ((i = r), (r = void 0)), n.consume(s), u;
	}
	function a(s) {
		return i || r || !l ? t(s) : e(s);
	}
}
function Ho(n, e) {
	let t = 0,
		r = 0;
	return i;
	function i(u) {
		return u === 40
			? (t++, n.consume(u), i)
			: u === 41 && r < t
				? l(u)
				: u === 33 ||
						u === 34 ||
						u === 38 ||
						u === 39 ||
						u === 41 ||
						u === 42 ||
						u === 44 ||
						u === 46 ||
						u === 58 ||
						u === 59 ||
						u === 60 ||
						u === 63 ||
						u === 93 ||
						u === 95 ||
						u === 126
					? n.check(xr, e, l)(u)
					: u === null || $(u) || An(u)
						? e(u)
						: (n.consume(u), i);
	}
	function l(u) {
		return u === 41 && r++, n.consume(u), i;
	}
}
function Uo(n, e, t) {
	return r;
	function r(o) {
		return o === 33 ||
			o === 34 ||
			o === 39 ||
			o === 41 ||
			o === 42 ||
			o === 44 ||
			o === 46 ||
			o === 58 ||
			o === 59 ||
			o === 63 ||
			o === 95 ||
			o === 126
			? (n.consume(o), r)
			: o === 38
				? (n.consume(o), l)
				: o === 93
					? (n.consume(o), i)
					: o === 60 || o === null || $(o) || An(o)
						? e(o)
						: t(o);
	}
	function i(o) {
		return o === null || o === 40 || o === 91 || $(o) || An(o) ? e(o) : r(o);
	}
	function l(o) {
		return J(o) ? u(o) : t(o);
	}
	function u(o) {
		return o === 59 ? (n.consume(o), r) : J(o) ? (n.consume(o), u) : t(o);
	}
}
function qo(n, e, t) {
	return r;
	function r(l) {
		return n.consume(l), i;
	}
	function i(l) {
		return Z(l) ? t(l) : e(l);
	}
}
function Sr(n) {
	return (
		n === null ||
		n === 40 ||
		n === 42 ||
		n === 95 ||
		n === 91 ||
		n === 93 ||
		n === 126 ||
		$(n)
	);
}
function Cr(n) {
	return !J(n);
}
function Er(n) {
	return !(n === 47 || Ie(n));
}
function Ie(n) {
	return n === 43 || n === 45 || n === 46 || n === 95 || Z(n);
}
function Ne(n) {
	let e = n.length,
		t = !1;
	for (; e--; ) {
		const r = n[e][1];
		if ((r.type === "labelLink" || r.type === "labelImage") && !r._balanced) {
			t = !0;
			break;
		}
		if (r._gfmAutolinkLiteralWalkedInto) {
			t = !1;
			break;
		}
	}
	return (
		n.length > 0 &&
			!t &&
			(n[n.length - 1][1]._gfmAutolinkLiteralWalkedInto = !0),
		t
	);
}
const Vo = { tokenize: Zo, partial: !0 };
function $o() {
	return {
		document: {
			91: { tokenize: Go, continuation: { tokenize: Yo }, exit: Ko },
		},
		text: {
			91: { tokenize: Xo },
			93: { add: "after", tokenize: Wo, resolveTo: Qo },
		},
	};
}
function Wo(n, e, t) {
	const r = this;
	let i = r.events.length;
	const l = r.parser.gfmFootnotes || (r.parser.gfmFootnotes = []);
	let u;
	for (; i--; ) {
		const a = r.events[i][1];
		if (a.type === "labelImage") {
			u = a;
			break;
		}
		if (
			a.type === "gfmFootnoteCall" ||
			a.type === "labelLink" ||
			a.type === "label" ||
			a.type === "image" ||
			a.type === "link"
		)
			break;
	}
	return o;
	function o(a) {
		if (!u || !u._balanced) return t(a);
		const s = fn(r.sliceSerialize({ start: u.end, end: r.now() }));
		return s.codePointAt(0) !== 94 || !l.includes(s.slice(1))
			? t(a)
			: (n.enter("gfmFootnoteCallLabelMarker"),
				n.consume(a),
				n.exit("gfmFootnoteCallLabelMarker"),
				e(a));
	}
}
function Qo(n, e) {
	let t = n.length;
	for (; t--; )
		if (n[t][1].type === "labelImage" && n[t][0] === "enter") {
			n[t][1];
			break;
		}
	(n[t + 1][1].type = "data"),
		(n[t + 3][1].type = "gfmFootnoteCallLabelMarker");
	const r = {
			type: "gfmFootnoteCall",
			start: Object.assign({}, n[t + 3][1].start),
			end: Object.assign({}, n[n.length - 1][1].end),
		},
		i = {
			type: "gfmFootnoteCallMarker",
			start: Object.assign({}, n[t + 3][1].end),
			end: Object.assign({}, n[t + 3][1].end),
		};
	i.end.column++, i.end.offset++, i.end._bufferIndex++;
	const l = {
			type: "gfmFootnoteCallString",
			start: Object.assign({}, i.end),
			end: Object.assign({}, n[n.length - 1][1].start),
		},
		u = {
			type: "chunkString",
			contentType: "string",
			start: Object.assign({}, l.start),
			end: Object.assign({}, l.end),
		},
		o = [
			n[t + 1],
			n[t + 2],
			["enter", r, e],
			n[t + 3],
			n[t + 4],
			["enter", i, e],
			["exit", i, e],
			["enter", l, e],
			["enter", u, e],
			["exit", u, e],
			["exit", l, e],
			n[n.length - 2],
			n[n.length - 1],
			["exit", r, e],
		];
	return n.splice(t, n.length - t + 1, ...o), n;
}
function Xo(n, e, t) {
	const r = this,
		i = r.parser.gfmFootnotes || (r.parser.gfmFootnotes = []);
	let l = 0,
		u;
	return o;
	function o(c) {
		return (
			n.enter("gfmFootnoteCall"),
			n.enter("gfmFootnoteCallLabelMarker"),
			n.consume(c),
			n.exit("gfmFootnoteCallLabelMarker"),
			a
		);
	}
	function a(c) {
		return c !== 94
			? t(c)
			: (n.enter("gfmFootnoteCallMarker"),
				n.consume(c),
				n.exit("gfmFootnoteCallMarker"),
				n.enter("gfmFootnoteCallString"),
				(n.enter("chunkString").contentType = "string"),
				s);
	}
	function s(c) {
		if (l > 999 || (c === 93 && !u) || c === null || c === 91 || $(c))
			return t(c);
		if (c === 93) {
			n.exit("chunkString");
			const p = n.exit("gfmFootnoteCallString");
			return i.includes(fn(r.sliceSerialize(p)))
				? (n.enter("gfmFootnoteCallLabelMarker"),
					n.consume(c),
					n.exit("gfmFootnoteCallLabelMarker"),
					n.exit("gfmFootnoteCall"),
					e)
				: t(c);
		}
		return $(c) || (u = !0), l++, n.consume(c), c === 92 ? f : s;
	}
	function f(c) {
		return c === 91 || c === 92 || c === 93 ? (n.consume(c), l++, s) : s(c);
	}
}
function Go(n, e, t) {
	const r = this,
		i = r.parser.gfmFootnotes || (r.parser.gfmFootnotes = []);
	let l,
		u = 0,
		o;
	return a;
	function a(d) {
		return (
			(n.enter("gfmFootnoteDefinition")._container = !0),
			n.enter("gfmFootnoteDefinitionLabel"),
			n.enter("gfmFootnoteDefinitionLabelMarker"),
			n.consume(d),
			n.exit("gfmFootnoteDefinitionLabelMarker"),
			s
		);
	}
	function s(d) {
		return d === 94
			? (n.enter("gfmFootnoteDefinitionMarker"),
				n.consume(d),
				n.exit("gfmFootnoteDefinitionMarker"),
				n.enter("gfmFootnoteDefinitionLabelString"),
				(n.enter("chunkString").contentType = "string"),
				f)
			: t(d);
	}
	function f(d) {
		if (u > 999 || (d === 93 && !o) || d === null || d === 91 || $(d))
			return t(d);
		if (d === 93) {
			n.exit("chunkString");
			const y = n.exit("gfmFootnoteDefinitionLabelString");
			return (
				(l = fn(r.sliceSerialize(y))),
				n.enter("gfmFootnoteDefinitionLabelMarker"),
				n.consume(d),
				n.exit("gfmFootnoteDefinitionLabelMarker"),
				n.exit("gfmFootnoteDefinitionLabel"),
				p
			);
		}
		return $(d) || (o = !0), u++, n.consume(d), d === 92 ? c : f;
	}
	function c(d) {
		return d === 91 || d === 92 || d === 93 ? (n.consume(d), u++, f) : f(d);
	}
	function p(d) {
		return d === 58
			? (n.enter("definitionMarker"),
				n.consume(d),
				n.exit("definitionMarker"),
				i.includes(l) || i.push(l),
				U(n, h, "gfmFootnoteDefinitionWhitespace"))
			: t(d);
	}
	function h(d) {
		return e(d);
	}
}
function Yo(n, e, t) {
	return n.check(Hn, e, n.attempt(Vo, e, t));
}
function Ko(n) {
	n.exit("gfmFootnoteDefinition");
}
function Zo(n, e, t) {
	const r = this;
	return U(n, i, "gfmFootnoteDefinitionIndent", 5);
	function i(l) {
		const u = r.events[r.events.length - 1];
		return u &&
			u[1].type === "gfmFootnoteDefinitionIndent" &&
			u[2].sliceSerialize(u[1], !0).length === 4
			? e(l)
			: t(l);
	}
}
function Jo(n) {
	let t = (n || {}).singleTilde;
	const r = { tokenize: l, resolveAll: i };
	return (
		t == null && (t = !0),
		{
			text: { 126: r },
			insideSpan: { null: [r] },
			attentionMarkers: { null: [126] },
		}
	);
	function i(u, o) {
		let a = -1;
		for (; ++a < u.length; )
			if (
				u[a][0] === "enter" &&
				u[a][1].type === "strikethroughSequenceTemporary" &&
				u[a][1]._close
			) {
				let s = a;
				for (; s--; )
					if (
						u[s][0] === "exit" &&
						u[s][1].type === "strikethroughSequenceTemporary" &&
						u[s][1]._open &&
						u[a][1].end.offset - u[a][1].start.offset ===
							u[s][1].end.offset - u[s][1].start.offset
					) {
						(u[a][1].type = "strikethroughSequence"),
							(u[s][1].type = "strikethroughSequence");
						const f = {
								type: "strikethrough",
								start: Object.assign({}, u[s][1].start),
								end: Object.assign({}, u[a][1].end),
							},
							c = {
								type: "strikethroughText",
								start: Object.assign({}, u[s][1].end),
								end: Object.assign({}, u[a][1].start),
							},
							p = [
								["enter", f, o],
								["enter", u[s][1], o],
								["exit", u[s][1], o],
								["enter", c, o],
							],
							h = o.parser.constructs.insideSpan.null;
						h && tn(p, p.length, 0, Zn(h, u.slice(s + 1, a), o)),
							tn(p, p.length, 0, [
								["exit", c, o],
								["enter", u[a][1], o],
								["exit", u[a][1], o],
								["exit", f, o],
							]),
							tn(u, s - 1, a - s + 3, p),
							(a = s + p.length - 2);
						break;
					}
			}
		for (a = -1; ++a < u.length; )
			u[a][1].type === "strikethroughSequenceTemporary" &&
				(u[a][1].type = "data");
		return u;
	}
	function l(u, o, a) {
		const s = this.previous,
			f = this.events;
		let c = 0;
		return p;
		function p(d) {
			return s === 126 && f[f.length - 1][1].type !== "characterEscape"
				? a(d)
				: (u.enter("strikethroughSequenceTemporary"), h(d));
		}
		function h(d) {
			const y = Gn(s);
			if (d === 126) return c > 1 ? a(d) : (u.consume(d), c++, h);
			if (c < 2 && !t) return a(d);
			const S = u.exit("strikethroughSequenceTemporary"),
				k = Gn(d);
			return (
				(S._open = !k || (k === 2 && !!y)),
				(S._close = !y || (y === 2 && !!k)),
				o(d)
			);
		}
	}
}
class na {
	constructor() {
		this.map = [];
	}
	add(e, t, r) {
		ea(this, e, t, r);
	}
	consume(e) {
		if ((this.map.sort((l, u) => l[0] - u[0]), this.map.length === 0)) return;
		let t = this.map.length;
		const r = [];
		for (; t > 0; )
			(t -= 1),
				r.push(e.slice(this.map[t][0] + this.map[t][1])),
				r.push(this.map[t][2]),
				(e.length = this.map[t][0]);
		r.push([...e]), (e.length = 0);
		let i = r.pop();
		for (; i; ) e.push(...i), (i = r.pop());
		this.map.length = 0;
	}
}
function ea(n, e, t, r) {
	let i = 0;
	if (!(t === 0 && r.length === 0)) {
		for (; i < n.map.length; ) {
			if (n.map[i][0] === e) {
				(n.map[i][1] += t), n.map[i][2].push(...r);
				return;
			}
			i += 1;
		}
		n.map.push([e, t, r]);
	}
}
function ta(n, e) {
	let t = !1;
	const r = [];
	for (; e < n.length; ) {
		const i = n[e];
		if (t) {
			if (i[0] === "enter")
				i[1].type === "tableContent" &&
					r.push(n[e + 1][1].type === "tableDelimiterMarker" ? "left" : "none");
			else if (i[1].type === "tableContent") {
				if (n[e - 1][1].type === "tableDelimiterMarker") {
					const l = r.length - 1;
					r[l] = r[l] === "left" ? "center" : "right";
				}
			} else if (i[1].type === "tableDelimiterRow") break;
		} else i[0] === "enter" && i[1].type === "tableDelimiterRow" && (t = !0);
		e += 1;
	}
	return r;
}
const ra = { flow: { null: { tokenize: ia, resolveAll: la } } };
function ia(n, e, t) {
	const r = this;
	let i = 0,
		l = 0,
		u;
	return o;
	function o(b) {
		let I = r.events.length - 1;
		for (; I > -1; ) {
			const P = r.events[I][1].type;
			if (P === "lineEnding" || P === "linePrefix") I--;
			else break;
		}
		const T = I > -1 ? r.events[I][1].type : null,
			O = T === "tableHead" || T === "tableRow" ? x : a;
		return O === x && r.parser.lazy[r.now().line] ? t(b) : O(b);
	}
	function a(b) {
		return n.enter("tableHead"), n.enter("tableRow"), s(b);
	}
	function s(b) {
		return b === 124 || ((u = !0), (l += 1)), f(b);
	}
	function f(b) {
		return b === null
			? t(b)
			: z(b)
				? l > 1
					? ((l = 0),
						(r.interrupt = !0),
						n.exit("tableRow"),
						n.enter("lineEnding"),
						n.consume(b),
						n.exit("lineEnding"),
						h)
					: t(b)
				: j(b)
					? U(n, f, "whitespace")(b)
					: ((l += 1),
						u && ((u = !1), (i += 1)),
						b === 124
							? (n.enter("tableCellDivider"),
								n.consume(b),
								n.exit("tableCellDivider"),
								(u = !0),
								f)
							: (n.enter("data"), c(b)));
	}
	function c(b) {
		return b === null || b === 124 || $(b)
			? (n.exit("data"), f(b))
			: (n.consume(b), b === 92 ? p : c);
	}
	function p(b) {
		return b === 92 || b === 124 ? (n.consume(b), c) : c(b);
	}
	function h(b) {
		return (
			(r.interrupt = !1),
			r.parser.lazy[r.now().line]
				? t(b)
				: (n.enter("tableDelimiterRow"),
					(u = !1),
					j(b)
						? U(
								n,
								d,
								"linePrefix",
								r.parser.constructs.disable.null.includes("codeIndented")
									? void 0
									: 4,
							)(b)
						: d(b))
		);
	}
	function d(b) {
		return b === 45 || b === 58
			? S(b)
			: b === 124
				? ((u = !0),
					n.enter("tableCellDivider"),
					n.consume(b),
					n.exit("tableCellDivider"),
					y)
				: R(b);
	}
	function y(b) {
		return j(b) ? U(n, S, "whitespace")(b) : S(b);
	}
	function S(b) {
		return b === 58
			? ((l += 1),
				(u = !0),
				n.enter("tableDelimiterMarker"),
				n.consume(b),
				n.exit("tableDelimiterMarker"),
				k)
			: b === 45
				? ((l += 1), k(b))
				: b === null || z(b)
					? L(b)
					: R(b);
	}
	function k(b) {
		return b === 45 ? (n.enter("tableDelimiterFiller"), A(b)) : R(b);
	}
	function A(b) {
		return b === 45
			? (n.consume(b), A)
			: b === 58
				? ((u = !0),
					n.exit("tableDelimiterFiller"),
					n.enter("tableDelimiterMarker"),
					n.consume(b),
					n.exit("tableDelimiterMarker"),
					C)
				: (n.exit("tableDelimiterFiller"), C(b));
	}
	function C(b) {
		return j(b) ? U(n, L, "whitespace")(b) : L(b);
	}
	function L(b) {
		return b === 124
			? d(b)
			: b === null || z(b)
				? !u || i !== l
					? R(b)
					: (n.exit("tableDelimiterRow"), n.exit("tableHead"), e(b))
				: R(b);
	}
	function R(b) {
		return t(b);
	}
	function x(b) {
		return n.enter("tableRow"), D(b);
	}
	function D(b) {
		return b === 124
			? (n.enter("tableCellDivider"),
				n.consume(b),
				n.exit("tableCellDivider"),
				D)
			: b === null || z(b)
				? (n.exit("tableRow"), e(b))
				: j(b)
					? U(n, D, "whitespace")(b)
					: (n.enter("data"), M(b));
	}
	function M(b) {
		return b === null || b === 124 || $(b)
			? (n.exit("data"), D(b))
			: (n.consume(b), b === 92 ? H : M);
	}
	function H(b) {
		return b === 92 || b === 124 ? (n.consume(b), M) : M(b);
	}
}
function la(n, e) {
	let t = -1,
		r = !0,
		i = 0,
		l = [0, 0, 0, 0],
		u = [0, 0, 0, 0],
		o = !1,
		a = 0,
		s,
		f,
		c;
	const p = new na();
	for (; ++t < n.length; ) {
		const h = n[t],
			d = h[1];
		h[0] === "enter"
			? d.type === "tableHead"
				? ((o = !1),
					a !== 0 && (At(p, e, a, s, f), (f = void 0), (a = 0)),
					(s = {
						type: "table",
						start: Object.assign({}, d.start),
						end: Object.assign({}, d.end),
					}),
					p.add(t, 0, [["enter", s, e]]))
				: d.type === "tableRow" || d.type === "tableDelimiterRow"
					? ((r = !0),
						(c = void 0),
						(l = [0, 0, 0, 0]),
						(u = [0, t + 1, 0, 0]),
						o &&
							((o = !1),
							(f = {
								type: "tableBody",
								start: Object.assign({}, d.start),
								end: Object.assign({}, d.end),
							}),
							p.add(t, 0, [["enter", f, e]])),
						(i = d.type === "tableDelimiterRow" ? 2 : f ? 3 : 1))
					: i &&
							(d.type === "data" ||
								d.type === "tableDelimiterMarker" ||
								d.type === "tableDelimiterFiller")
						? ((r = !1),
							u[2] === 0 &&
								(l[1] !== 0 &&
									((u[0] = u[1]),
									(c = Wn(p, e, l, i, void 0, c)),
									(l = [0, 0, 0, 0])),
								(u[2] = t)))
						: d.type === "tableCellDivider" &&
							(r
								? (r = !1)
								: (l[1] !== 0 &&
										((u[0] = u[1]), (c = Wn(p, e, l, i, void 0, c))),
									(l = u),
									(u = [l[1], t, 0, 0])))
			: d.type === "tableHead"
				? ((o = !0), (a = t))
				: d.type === "tableRow" || d.type === "tableDelimiterRow"
					? ((a = t),
						l[1] !== 0
							? ((u[0] = u[1]), (c = Wn(p, e, l, i, t, c)))
							: u[1] !== 0 && (c = Wn(p, e, u, i, t, c)),
						(i = 0))
					: i &&
						(d.type === "data" ||
							d.type === "tableDelimiterMarker" ||
							d.type === "tableDelimiterFiller") &&
						(u[3] = t);
	}
	for (
		a !== 0 && At(p, e, a, s, f), p.consume(e.events), t = -1;
		++t < e.events.length;
	) {
		const h = e.events[t];
		h[0] === "enter" &&
			h[1].type === "table" &&
			(h[1]._align = ta(e.events, t));
	}
	return n;
}
function Wn(n, e, t, r, i, l) {
	const u = r === 1 ? "tableHeader" : r === 2 ? "tableDelimiter" : "tableData",
		o = "tableContent";
	t[0] !== 0 &&
		((l.end = Object.assign({}, Ln(e.events, t[0]))),
		n.add(t[0], 0, [["exit", l, e]]));
	const a = Ln(e.events, t[1]);
	if (
		((l = { type: u, start: Object.assign({}, a), end: Object.assign({}, a) }),
		n.add(t[1], 0, [["enter", l, e]]),
		t[2] !== 0)
	) {
		const s = Ln(e.events, t[2]),
			f = Ln(e.events, t[3]),
			c = { type: o, start: Object.assign({}, s), end: Object.assign({}, f) };
		if ((n.add(t[2], 0, [["enter", c, e]]), r !== 2)) {
			const p = e.events[t[2]],
				h = e.events[t[3]];
			if (
				((p[1].end = Object.assign({}, h[1].end)),
				(p[1].type = "chunkText"),
				(p[1].contentType = "text"),
				t[3] > t[2] + 1)
			) {
				const d = t[2] + 1,
					y = t[3] - t[2] - 1;
				n.add(d, y, []);
			}
		}
		n.add(t[3] + 1, 0, [["exit", c, e]]);
	}
	return (
		i !== void 0 &&
			((l.end = Object.assign({}, Ln(e.events, i))),
			n.add(i, 0, [["exit", l, e]]),
			(l = void 0)),
		l
	);
}
function At(n, e, t, r, i) {
	const l = [],
		u = Ln(e.events, t);
	i && ((i.end = Object.assign({}, u)), l.push(["exit", i, e])),
		(r.end = Object.assign({}, u)),
		l.push(["exit", r, e]),
		n.add(t + 1, 0, l);
}
function Ln(n, e) {
	const t = n[e],
		r = t[0] === "enter" ? "start" : "end";
	return t[1][r];
}
const ua = { tokenize: aa },
	oa = { text: { 91: ua } };
function aa(n, e, t) {
	const r = this;
	return i;
	function i(a) {
		return r.previous !== null || !r._gfmTasklistFirstContentOfListItem
			? t(a)
			: (n.enter("taskListCheck"),
				n.enter("taskListCheckMarker"),
				n.consume(a),
				n.exit("taskListCheckMarker"),
				l);
	}
	function l(a) {
		return $(a)
			? (n.enter("taskListCheckValueUnchecked"),
				n.consume(a),
				n.exit("taskListCheckValueUnchecked"),
				u)
			: a === 88 || a === 120
				? (n.enter("taskListCheckValueChecked"),
					n.consume(a),
					n.exit("taskListCheckValueChecked"),
					u)
				: t(a);
	}
	function u(a) {
		return a === 93
			? (n.enter("taskListCheckMarker"),
				n.consume(a),
				n.exit("taskListCheckMarker"),
				n.exit("taskListCheck"),
				o)
			: t(a);
	}
	function o(a) {
		return z(a) ? e(a) : j(a) ? n.check({ tokenize: sa }, e, t)(a) : t(a);
	}
}
function sa(n, e, t) {
	return U(n, r, "whitespace");
	function r(i) {
		return i === null ? t(i) : e(i);
	}
}
function ca(n) {
	return jt([vo, $o(), Jo(n), ra, oa]);
}
function Ft(n, e) {
	const t = String(n);
	if (typeof e != "string") throw new TypeError("Expected character");
	let r = 0,
		i = t.indexOf(e);
	for (; i !== -1; ) r++, (i = t.indexOf(e, i + e.length));
	return r;
}
function fa(n) {
	if (typeof n != "string") throw new TypeError("Expected a string");
	return n.replace(/[|\\{}()[\]^$+*?.]/g, "\\$&").replace(/-/g, "\\x2d");
}
const ha = {}.hasOwnProperty,
	pa = (n, e, t, r) => {
		let i, l;
		typeof e == "string" || e instanceof RegExp
			? ((l = [[e, t]]), (i = r))
			: ((l = e), (i = t)),
			i || (i = {});
		const u = Re(i.ignore || []),
			o = ma(l);
		let a = -1;
		for (; ++a < o.length; ) ir(n, "text", s);
		return n;
		function s(c, p) {
			let h = -1,
				d;
			for (; ++h < p.length; ) {
				const y = p[h];
				if (u(y, d ? d.children.indexOf(y) : void 0, d)) return;
				d = y;
			}
			if (d) return f(c, p);
		}
		function f(c, p) {
			const h = p[p.length - 1],
				d = o[a][0],
				y = o[a][1];
			let S = 0;
			const k = h.children.indexOf(c);
			let A = !1,
				C = [];
			d.lastIndex = 0;
			let L = d.exec(c.value);
			for (; L; ) {
				const R = L.index,
					x = { index: L.index, input: L.input, stack: [...p, c] };
				let D = y(...L, x);
				if (
					(typeof D == "string" &&
						(D = D.length > 0 ? { type: "text", value: D } : void 0),
					D !== !1 &&
						(S !== R && C.push({ type: "text", value: c.value.slice(S, R) }),
						Array.isArray(D) ? C.push(...D) : D && C.push(D),
						(S = R + L[0].length),
						(A = !0)),
					!d.global)
				)
					break;
				L = d.exec(c.value);
			}
			return (
				A
					? (S < c.value.length &&
							C.push({ type: "text", value: c.value.slice(S) }),
						h.children.splice(k, 1, ...C))
					: (C = [c]),
				k + C.length
			);
		}
	};
function ma(n) {
	const e = [];
	if (typeof n != "object")
		throw new TypeError("Expected array or object as schema");
	if (Array.isArray(n)) {
		let t = -1;
		for (; ++t < n.length; ) e.push([Dt(n[t][0]), It(n[t][1])]);
	} else {
		let t;
		for (t in n) ha.call(n, t) && e.push([Dt(t), It(n[t])]);
	}
	return e;
}
function Dt(n) {
	return typeof n == "string" ? new RegExp(fa(n), "g") : n;
}
function It(n) {
	return typeof n == "function" ? n : () => n;
}
const de = "phrasing",
	ye = ["autolink", "link", "image", "label"],
	ga = {
		transforms: [Sa],
		enter: {
			literalAutolink: ya,
			literalAutolinkEmail: ke,
			literalAutolinkHttp: ke,
			literalAutolinkWww: ke,
		},
		exit: {
			literalAutolink: wa,
			literalAutolinkEmail: ba,
			literalAutolinkHttp: ka,
			literalAutolinkWww: xa,
		},
	},
	da = {
		unsafe: [
			{
				character: "@",
				before: "[+\\-.\\w]",
				after: "[\\-.\\w]",
				inConstruct: de,
				notInConstruct: ye,
			},
			{
				character: ".",
				before: "[Ww]",
				after: "[\\-.\\w]",
				inConstruct: de,
				notInConstruct: ye,
			},
			{
				character: ":",
				before: "[ps]",
				after: "\\/",
				inConstruct: de,
				notInConstruct: ye,
			},
		],
	};
function ya(n) {
	this.enter({ type: "link", title: null, url: "", children: [] }, n);
}
function ke(n) {
	this.config.enter.autolinkProtocol.call(this, n);
}
function ka(n) {
	this.config.exit.autolinkProtocol.call(this, n);
}
function xa(n) {
	this.config.exit.data.call(this, n);
	const e = this.stack[this.stack.length - 1];
	e.url = "http://" + this.sliceSerialize(n);
}
function ba(n) {
	this.config.exit.autolinkEmail.call(this, n);
}
function wa(n) {
	this.exit(n);
}
function Sa(n) {
	pa(
		n,
		[
			[/(https?:\/\/|www(?=\.))([-.\w]+)([^ \t\r\n]*)/gi, Ca],
			[/([-.\w+]+)@([-\w]+(?:\.[-\w]+)+)/g, Ea],
		],
		{ ignore: ["link", "linkReference"] },
	);
}
function Ca(n, e, t, r, i) {
	let l = "";
	if (
		!Ar(i) ||
		(/^w/i.test(e) && ((t = e + t), (e = ""), (l = "http://")), !Aa(t))
	)
		return !1;
	const u = Fa(t + r);
	if (!u[0]) return !1;
	const o = {
		type: "link",
		title: null,
		url: l + e + u[0],
		children: [{ type: "text", value: e + u[0] }],
	};
	return u[1] ? [o, { type: "text", value: u[1] }] : o;
}
function Ea(n, e, t, r) {
	return !Ar(r, !0) || /[-\d_]$/.test(t)
		? !1
		: {
				type: "link",
				title: null,
				url: "mailto:" + e + "@" + t,
				children: [{ type: "text", value: e + "@" + t }],
			};
}
function Aa(n) {
	const e = n.split(".");
	return !(
		e.length < 2 ||
		(e[e.length - 1] &&
			(/_/.test(e[e.length - 1]) || !/[a-zA-Z\d]/.test(e[e.length - 1]))) ||
		(e[e.length - 2] &&
			(/_/.test(e[e.length - 2]) || !/[a-zA-Z\d]/.test(e[e.length - 2])))
	);
}
function Fa(n) {
	const e = /[!"&'),.:;<>?\]}]+$/.exec(n);
	if (!e) return [n, void 0];
	n = n.slice(0, e.index);
	let t = e[0],
		r = t.indexOf(")");
	const i = Ft(n, "(");
	let l = Ft(n, ")");
	for (; r !== -1 && i > l; )
		(n += t.slice(0, r + 1)), (t = t.slice(r + 1)), (r = t.indexOf(")")), l++;
	return [n, t];
}
function Ar(n, e) {
	const t = n.input.charCodeAt(n.index - 1);
	return (n.index === 0 || An(t) || Kn(t)) && (!e || t !== 47);
}
function Fr(n) {
	return n.label || !n.identifier ? n.label || "" : Kt(n.identifier);
}
function Da(n, e, t) {
	const r = e.indexStack,
		i = n.children || [],
		l = e.createTracker(t),
		u = [];
	let o = -1;
	for (r.push(-1); ++o < i.length; ) {
		const a = i[o];
		(r[r.length - 1] = o),
			u.push(
				l.move(
					e.handle(a, n, e, {
						before: `
`,
						after: `
`,
						...l.current(),
					}),
				),
			),
			a.type !== "list" && (e.bulletLastUsed = void 0),
			o < i.length - 1 && u.push(l.move(Ia(a, i[o + 1], n, e)));
	}
	return r.pop(), u.join("");
}
function Ia(n, e, t, r) {
	let i = r.join.length;
	for (; i--; ) {
		const l = r.join[i](n, e, t, r);
		if (l === !0 || l === 1) break;
		if (typeof l == "number")
			return `
`.repeat(1 + l);
		if (l === !1)
			return `

<!---->

`;
	}
	return `

`;
}
const Ta = /\r?\n|\r/g;
function Pa(n, e) {
	const t = [];
	let r = 0,
		i = 0,
		l;
	for (; (l = Ta.exec(n)); )
		u(n.slice(r, l.index)), t.push(l[0]), (r = l.index + l[0].length), i++;
	return u(n.slice(r)), t.join("");
	function u(o) {
		t.push(e(o, i, !o));
	}
}
function Dr(n) {
	if (!n._compiled) {
		const e =
			(n.atBreak ? "[\\r\\n][\\t ]*" : "") +
			(n.before ? "(?:" + n.before + ")" : "");
		n._compiled = new RegExp(
			(e ? "(" + e + ")" : "") +
				(/[|\\{}()[\]^$+*?.-]/.test(n.character) ? "\\" : "") +
				n.character +
				(n.after ? "(?:" + n.after + ")" : ""),
			"g",
		);
	}
	return n._compiled;
}
function La(n, e) {
	return Tt(n, e.inConstruct, !0) && !Tt(n, e.notInConstruct, !1);
}
function Tt(n, e, t) {
	if ((typeof e == "string" && (e = [e]), !e || e.length === 0)) return t;
	let r = -1;
	for (; ++r < e.length; ) if (n.includes(e[r])) return !0;
	return !1;
}
function Ir(n, e, t) {
	const r = (t.before || "") + (e || "") + (t.after || ""),
		i = [],
		l = [],
		u = {};
	let o = -1;
	for (; ++o < n.unsafe.length; ) {
		const f = n.unsafe[o];
		if (!La(n.stack, f)) continue;
		const c = Dr(f);
		let p;
		for (; (p = c.exec(r)); ) {
			const h = "before" in f || !!f.atBreak,
				d = "after" in f,
				y = p.index + (h ? p[1].length : 0);
			i.includes(y)
				? (u[y].before && !h && (u[y].before = !1),
					u[y].after && !d && (u[y].after = !1))
				: (i.push(y), (u[y] = { before: h, after: d }));
		}
	}
	i.sort(Oa);
	let a = t.before ? t.before.length : 0;
	const s = r.length - (t.after ? t.after.length : 0);
	for (o = -1; ++o < i.length; ) {
		const f = i[o];
		f < a ||
			f >= s ||
			(f + 1 < s &&
				i[o + 1] === f + 1 &&
				u[f].after &&
				!u[f + 1].before &&
				!u[f + 1].after) ||
			(i[o - 1] === f - 1 &&
				u[f].before &&
				!u[f - 1].before &&
				!u[f - 1].after) ||
			(a !== f && l.push(Pt(r.slice(a, f), "\\")),
			(a = f),
			/[!-/:-@[-`{-~]/.test(r.charAt(f)) &&
			(!t.encode || !t.encode.includes(r.charAt(f)))
				? l.push("\\")
				: (l.push("&#x" + r.charCodeAt(f).toString(16).toUpperCase() + ";"),
					a++));
	}
	return l.push(Pt(r.slice(a, s), t.after)), l.join("");
}
function Oa(n, e) {
	return n - e;
}
function Pt(n, e) {
	const t = /\\(?=[!-/:-@[-`{-~])/g,
		r = [],
		i = [],
		l = n + e;
	let u = -1,
		o = 0,
		a;
	for (; (a = t.exec(l)); ) r.push(a.index);
	for (; ++u < r.length; )
		o !== r[u] && i.push(n.slice(o, r[u])), i.push("\\"), (o = r[u]);
	return i.push(n.slice(o)), i.join("");
}
function ne(n) {
	const e = n || {},
		t = e.now || {};
	let r = e.lineShift || 0,
		i = t.line || 1,
		l = t.column || 1;
	return { move: a, current: u, shift: o };
	function u() {
		return { now: { line: i, column: l }, lineShift: r };
	}
	function o(s) {
		r += s;
	}
	function a(s) {
		const f = s || "",
			c = f.split(/\r?\n|\r/g),
			p = c[c.length - 1];
		return (
			(i += c.length - 1),
			(l = c.length === 1 ? l + p.length : 1 + p.length + r),
			f
		);
	}
}
Tr.peek = qa;
function za() {
	return {
		enter: {
			gfmFootnoteDefinition: va,
			gfmFootnoteDefinitionLabelString: Ba,
			gfmFootnoteCall: _a,
			gfmFootnoteCallString: ja,
		},
		exit: {
			gfmFootnoteDefinition: Na,
			gfmFootnoteDefinitionLabelString: Ma,
			gfmFootnoteCall: Ua,
			gfmFootnoteCallString: Ha,
		},
	};
}
function Ra() {
	return {
		unsafe: [
			{ character: "[", inConstruct: ["phrasing", "label", "reference"] },
		],
		handlers: { footnoteDefinition: Va, footnoteReference: Tr },
	};
}
function va(n) {
	this.enter(
		{ type: "footnoteDefinition", identifier: "", label: "", children: [] },
		n,
	);
}
function Ba() {
	this.buffer();
}
function Ma(n) {
	const e = this.resume(),
		t = this.stack[this.stack.length - 1];
	(t.label = e), (t.identifier = fn(this.sliceSerialize(n)).toLowerCase());
}
function Na(n) {
	this.exit(n);
}
function _a(n) {
	this.enter({ type: "footnoteReference", identifier: "", label: "" }, n);
}
function ja() {
	this.buffer();
}
function Ha(n) {
	const e = this.resume(),
		t = this.stack[this.stack.length - 1];
	(t.label = e), (t.identifier = fn(this.sliceSerialize(n)).toLowerCase());
}
function Ua(n) {
	this.exit(n);
}
function Tr(n, e, t, r) {
	const i = ne(r);
	let l = i.move("[^");
	const u = t.enter("footnoteReference"),
		o = t.enter("reference");
	return (
		(l += i.move(Ir(t, Fr(n), { ...i.current(), before: l, after: "]" }))),
		o(),
		u(),
		(l += i.move("]")),
		l
	);
}
function qa() {
	return "[";
}
function Va(n, e, t, r) {
	const i = ne(r);
	let l = i.move("[^");
	const u = t.enter("footnoteDefinition"),
		o = t.enter("label");
	return (
		(l += i.move(Ir(t, Fr(n), { ...i.current(), before: l, after: "]" }))),
		o(),
		(l += i.move("]:" + (n.children && n.children.length > 0 ? " " : ""))),
		i.shift(4),
		(l += i.move(Pa(Da(n, t, i.current()), $a))),
		u(),
		l
	);
}
function $a(n, e, t) {
	return e === 0 ? n : (t ? "" : "    ") + n;
}
function Pr(n, e, t) {
	const r = e.indexStack,
		i = n.children || [],
		l = [];
	let u = -1,
		o = t.before;
	r.push(-1);
	let a = e.createTracker(t);
	for (; ++u < i.length; ) {
		const s = i[u];
		let f;
		if (((r[r.length - 1] = u), u + 1 < i.length)) {
			let c = e.handle.handlers[i[u + 1].type];
			c && c.peek && (c = c.peek),
				(f = c
					? c(i[u + 1], n, e, { before: "", after: "", ...a.current() }).charAt(
							0,
						)
					: "");
		} else f = t.after;
		l.length > 0 &&
			(o === "\r" ||
				o ===
					`
`) &&
			s.type === "html" &&
			((l[l.length - 1] = l[l.length - 1].replace(/(\r?\n|\r)$/, " ")),
			(o = " "),
			(a = e.createTracker(t)),
			a.move(l.join(""))),
			l.push(
				a.move(e.handle(s, n, e, { ...a.current(), before: o, after: f })),
			),
			(o = l[l.length - 1].slice(-1));
	}
	return r.pop(), l.join("");
}
const Wa = [
	"autolink",
	"destinationLiteral",
	"destinationRaw",
	"reference",
	"titleQuote",
	"titleApostrophe",
];
Lr.peek = Ka;
const Qa = {
		canContainEols: ["delete"],
		enter: { strikethrough: Ga },
		exit: { strikethrough: Ya },
	},
	Xa = {
		unsafe: [{ character: "~", inConstruct: "phrasing", notInConstruct: Wa }],
		handlers: { delete: Lr },
	};
function Ga(n) {
	this.enter({ type: "delete", children: [] }, n);
}
function Ya(n) {
	this.exit(n);
}
function Lr(n, e, t, r) {
	const i = ne(r),
		l = t.enter("strikethrough");
	let u = i.move("~~");
	return (
		(u += Pr(n, t, { ...i.current(), before: u, after: "~" })),
		(u += i.move("~~")),
		l(),
		u
	);
}
function Ka() {
	return "~";
}
Or.peek = Za;
function Or(n, e, t) {
	let r = n.value || "",
		i = "`",
		l = -1;
	for (; new RegExp("(^|[^`])" + i + "([^`]|$)").test(r); ) i += "`";
	for (
		/[^ \r\n]/.test(r) &&
		((/^[ \r\n]/.test(r) && /[ \r\n]$/.test(r)) || /^`|`$/.test(r)) &&
		(r = " " + r + " ");
		++l < t.unsafe.length;
	) {
		const u = t.unsafe[l],
			o = Dr(u);
		let a;
		if (u.atBreak)
			for (; (a = o.exec(r)); ) {
				let s = a.index;
				r.charCodeAt(s) === 10 && r.charCodeAt(s - 1) === 13 && s--,
					(r = r.slice(0, s) + " " + r.slice(a.index + 1));
			}
	}
	return i + r + i;
}
function Za() {
	return "`";
}
function Ja(n, e = {}) {
	const t = (e.align || []).concat(),
		r = e.stringLength || es,
		i = [],
		l = [],
		u = [],
		o = [];
	let a = 0,
		s = -1;
	for (; ++s < n.length; ) {
		const d = [],
			y = [];
		let S = -1;
		for (n[s].length > a && (a = n[s].length); ++S < n[s].length; ) {
			const k = ns(n[s][S]);
			if (e.alignDelimiters !== !1) {
				const A = r(k);
				(y[S] = A), (o[S] === void 0 || A > o[S]) && (o[S] = A);
			}
			d.push(k);
		}
		(l[s] = d), (u[s] = y);
	}
	let f = -1;
	if (typeof t == "object" && "length" in t) for (; ++f < a; ) i[f] = Lt(t[f]);
	else {
		const d = Lt(t);
		for (; ++f < a; ) i[f] = d;
	}
	f = -1;
	const c = [],
		p = [];
	for (; ++f < a; ) {
		const d = i[f];
		let y = "",
			S = "";
		d === 99
			? ((y = ":"), (S = ":"))
			: d === 108
				? (y = ":")
				: d === 114 && (S = ":");
		let k =
			e.alignDelimiters === !1 ? 1 : Math.max(1, o[f] - y.length - S.length);
		const A = y + "-".repeat(k) + S;
		e.alignDelimiters !== !1 &&
			((k = y.length + k + S.length), k > o[f] && (o[f] = k), (p[f] = k)),
			(c[f] = A);
	}
	l.splice(1, 0, c), u.splice(1, 0, p), (s = -1);
	const h = [];
	for (; ++s < l.length; ) {
		const d = l[s],
			y = u[s];
		f = -1;
		const S = [];
		for (; ++f < a; ) {
			const k = d[f] || "";
			let A = "",
				C = "";
			if (e.alignDelimiters !== !1) {
				const L = o[f] - (y[f] || 0),
					R = i[f];
				R === 114
					? (A = " ".repeat(L))
					: R === 99
						? L % 2
							? ((A = " ".repeat(L / 2 + 0.5)), (C = " ".repeat(L / 2 - 0.5)))
							: ((A = " ".repeat(L / 2)), (C = A))
						: (C = " ".repeat(L));
			}
			e.delimiterStart !== !1 && !f && S.push("|"),
				e.padding !== !1 &&
					!(e.alignDelimiters === !1 && k === "") &&
					(e.delimiterStart !== !1 || f) &&
					S.push(" "),
				e.alignDelimiters !== !1 && S.push(A),
				S.push(k),
				e.alignDelimiters !== !1 && S.push(C),
				e.padding !== !1 && S.push(" "),
				(e.delimiterEnd !== !1 || f !== a - 1) && S.push("|");
		}
		h.push(e.delimiterEnd === !1 ? S.join("").replace(/ +$/, "") : S.join(""));
	}
	return h.join(`
`);
}
function ns(n) {
	return n == null ? "" : String(n);
}
function es(n) {
	return n.length;
}
function Lt(n) {
	const e = typeof n == "string" ? n.codePointAt(0) : 0;
	return e === 67 || e === 99
		? 99
		: e === 76 || e === 108
			? 108
			: e === 82 || e === 114
				? 114
				: 0;
}
const ts = {
	enter: { table: rs, tableData: Ot, tableHeader: Ot, tableRow: ls },
	exit: {
		codeText: us,
		table: is,
		tableData: xe,
		tableHeader: xe,
		tableRow: xe,
	},
};
function rs(n) {
	const e = n._align;
	this.enter(
		{
			type: "table",
			align: e.map((t) => (t === "none" ? null : t)),
			children: [],
		},
		n,
	),
		this.setData("inTable", !0);
}
function is(n) {
	this.exit(n), this.setData("inTable");
}
function ls(n) {
	this.enter({ type: "tableRow", children: [] }, n);
}
function xe(n) {
	this.exit(n);
}
function Ot(n) {
	this.enter({ type: "tableCell", children: [] }, n);
}
function us(n) {
	let e = this.resume();
	this.getData("inTable") && (e = e.replace(/\\([\\|])/g, os));
	const t = this.stack[this.stack.length - 1];
	(t.value = e), this.exit(n);
}
function os(n, e) {
	return e === "|" ? e : n;
}
function as(n) {
	const e = n || {},
		t = e.tableCellPadding,
		r = e.tablePipeAlign,
		i = e.stringLength,
		l = t ? " " : "|";
	return {
		unsafe: [
			{ character: "\r", inConstruct: "tableCell" },
			{
				character: `
`,
				inConstruct: "tableCell",
			},
			{ atBreak: !0, character: "|", after: "[	 :-]" },
			{ character: "|", inConstruct: "tableCell" },
			{ atBreak: !0, character: ":", after: "-" },
			{ atBreak: !0, character: "-", after: "[:|-]" },
		],
		handlers: { table: u, tableRow: o, tableCell: a, inlineCode: p },
	};
	function u(h, d, y, S) {
		return s(f(h, y, S), h.align);
	}
	function o(h, d, y, S) {
		const k = c(h, y, S),
			A = s([k]);
		return A.slice(
			0,
			A.indexOf(`
`),
		);
	}
	function a(h, d, y, S) {
		const k = y.enter("tableCell"),
			A = y.enter("phrasing"),
			C = Pr(h, y, { ...S, before: l, after: l });
		return A(), k(), C;
	}
	function s(h, d) {
		return Ja(h, { align: d, alignDelimiters: r, padding: t, stringLength: i });
	}
	function f(h, d, y) {
		const S = h.children;
		let k = -1;
		const A = [],
			C = d.enter("table");
		for (; ++k < S.length; ) A[k] = c(S[k], d, y);
		return C(), A;
	}
	function c(h, d, y) {
		const S = h.children;
		let k = -1;
		const A = [],
			C = d.enter("tableRow");
		for (; ++k < S.length; ) A[k] = a(S[k], h, d, y);
		return C(), A;
	}
	function p(h, d, y) {
		let S = Or(h, d, y);
		return y.stack.includes("tableCell") && (S = S.replace(/\|/g, "\\$&")), S;
	}
}
function ss(n) {
	const e = n.options.bullet || "*";
	if (e !== "*" && e !== "+" && e !== "-")
		throw new Error(
			"Cannot serialize items with `" +
				e +
				"` for `options.bullet`, expected `*`, `+`, or `-`",
		);
	return e;
}
function cs(n) {
	const e = n.options.listItemIndent || "tab";
	if (e === 1 || e === "1") return "one";
	if (e !== "tab" && e !== "one" && e !== "mixed")
		throw new Error(
			"Cannot serialize items with `" +
				e +
				"` for `options.listItemIndent`, expected `tab`, `one`, or `mixed`",
		);
	return e;
}
function fs(n, e, t, r) {
	const i = cs(t);
	let l = t.bulletCurrent || ss(t);
	e &&
		e.type === "list" &&
		e.ordered &&
		(l =
			(typeof e.start == "number" && e.start > -1 ? e.start : 1) +
			(t.options.incrementListMarker === !1 ? 0 : e.children.indexOf(n)) +
			l);
	let u = l.length + 1;
	(i === "tab" ||
		(i === "mixed" && ((e && e.type === "list" && e.spread) || n.spread))) &&
		(u = Math.ceil(u / 4) * 4);
	const o = t.createTracker(r);
	o.move(l + " ".repeat(u - l.length)), o.shift(u);
	const a = t.enter("listItem"),
		s = t.indentLines(t.containerFlow(n, o.current()), f);
	return a(), s;
	function f(c, p, h) {
		return p
			? (h ? "" : " ".repeat(u)) + c
			: (h ? l : l + " ".repeat(u - l.length)) + c;
	}
}
const hs = {
		exit: {
			taskListCheckValueChecked: zt,
			taskListCheckValueUnchecked: zt,
			paragraph: ms,
		},
	},
	ps = {
		unsafe: [{ atBreak: !0, character: "-", after: "[:|-]" }],
		handlers: { listItem: gs },
	};
function zt(n) {
	const e = this.stack[this.stack.length - 2];
	e.checked = n.type === "taskListCheckValueChecked";
}
function ms(n) {
	const e = this.stack[this.stack.length - 2];
	if (e && e.type === "listItem" && typeof e.checked == "boolean") {
		const t = this.stack[this.stack.length - 1],
			r = t.children[0];
		if (r && r.type === "text") {
			const i = e.children;
			let l = -1,
				u;
			for (; ++l < i.length; ) {
				const o = i[l];
				if (o.type === "paragraph") {
					u = o;
					break;
				}
			}
			u === t &&
				((r.value = r.value.slice(1)),
				r.value.length === 0
					? t.children.shift()
					: t.position &&
						r.position &&
						typeof r.position.start.offset == "number" &&
						(r.position.start.column++,
						r.position.start.offset++,
						(t.position.start = Object.assign({}, r.position.start))));
		}
	}
	this.exit(n);
}
function gs(n, e, t, r) {
	const i = n.children[0],
		l = typeof n.checked == "boolean" && i && i.type === "paragraph",
		u = "[" + (n.checked ? "x" : " ") + "] ",
		o = ne(r);
	l && o.move(u);
	let a = fs(n, e, t, { ...r, ...o.current() });
	return l && (a = a.replace(/^(?:[*+-]|\d+\.)([\r\n]| {1,3})/, s)), a;
	function s(f) {
		return f + u;
	}
}
function ds() {
	return [ga, za(), Qa, ts, hs];
}
function ys(n) {
	return { extensions: [da, Ra(), Xa, as(n), ps] };
}
function ks(n = {}) {
	const e = this.data();
	t("micromarkExtensions", ca(n)),
		t("fromMarkdownExtensions", ds()),
		t("toMarkdownExtensions", ys(n));
	function t(r, i) {
		(e[r] ? e[r] : (e[r] = [])).push(i);
	}
}
function ws({ markdown: n }) {
	return Rt.jsx(dr, {
		components: { p: En.Fragment, a: xs },
		remarkPlugins: [ks],
		children: n,
	});
}
function xs(n) {
	return Rt.jsx("a", {
		...n,
		onClick: (e) => {
			e.preventDefault(), qr(n.href);
		},
	});
}
export { ws as ReactMarkdownGfm, ws as default };

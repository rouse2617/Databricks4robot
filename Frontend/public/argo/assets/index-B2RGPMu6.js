import { e as h, m } from "./editor.api-DXaA56bm.js";
import { p as n, r as y } from "./index-DXdSO-bO.js";
function M(r) {
	return /^\d+$/.test(r) ? "".concat(r, "px") : r;
}
function f() {}
var w = (() => {
		var r = (o, e) => (
			(r =
				Object.setPrototypeOf ||
				({ __proto__: [] } instanceof Array &&
					((t, i) => {
						t.__proto__ = i;
					})) ||
				((t, i) => {
					for (var a in i) Object.hasOwn(i, a) && (t[a] = i[a]);
				})),
			r(o, e)
		);
		return (o, e) => {
			if (typeof e != "function" && e !== null)
				throw new TypeError(
					"Class extends value " + String(e) + " is not a constructor or null",
				);
			r(o, e);
			function t() {
				this.constructor = o;
			}
			o.prototype =
				e === null ? Object.create(e) : ((t.prototype = e.prototype), new t());
		};
	})(),
	g = function () {
		return (
			(g =
				Object.assign ||
				function (r) {
					for (var o, e = 1, t = arguments.length; e < t; e++) {
						o = arguments[e];
						for (var i in o) Object.hasOwn(o, i) && (r[i] = o[i]);
					}
					return r;
				}),
			g.apply(this, arguments)
		);
	},
	U = ((r) => {
		w(o, r);
		function o(e) {
			var t = r.call(this, e) || this;
			return (
				(t.assignRef = (i) => {
					t.containerElement = i;
				}),
				(t.containerElement = void 0),
				t
			);
		}
		return (
			(o.prototype.componentDidMount = function () {
				this.initMonaco();
			}),
			(o.prototype.componentDidUpdate = function (e) {
				var t = this.props,
					i = t.language,
					a = t.theme,
					s = t.height,
					l = t.options,
					u = t.width,
					_ = t.className,
					v = this.editor.getModel(),
					c = v.original,
					d = v.modified;
				this.props.original !== c.getValue() && c.setValue(this.props.original),
					this.props.value != null &&
						this.props.value !== d.getValue() &&
						((this.__prevent_trigger_change_event = !0),
						this.editor.getModifiedEditor().pushUndoStop(),
						d.pushEditOperations(
							[],
							[{ range: d.getFullModelRange(), text: this.props.value }],
						),
						this.editor.getModifiedEditor().pushUndoStop(),
						(this.__prevent_trigger_change_event = !1)),
					e.language !== i &&
						(h.setModelLanguage(c, i), h.setModelLanguage(d, i)),
					e.theme !== a && h.setTheme(a),
					this.editor &&
						(u !== e.width || s !== e.height) &&
						this.editor.layout(),
					e.options !== l &&
						this.editor.updateOptions(
							g(g({}, _ ? { extraEditorClassName: _ } : {}), l),
						);
			}),
			(o.prototype.componentWillUnmount = function () {
				this.destroyMonaco();
			}),
			(o.prototype.editorWillMount = function () {
				var e = this.props.editorWillMount,
					t = e(m);
				return t || {};
			}),
			(o.prototype.editorDidMount = function (e) {
				this.props.editorDidMount(e, m);
				var i = e.getModel().modified;
				this._subscription = i.onDidChangeContent((a) => {
					this.__prevent_trigger_change_event ||
						this.props.onChange(i.getValue(), a);
				});
			}),
			(o.prototype.editorWillUnmount = function (e) {
				var t = this.props.editorWillUnmount;
				t(e, m);
			}),
			(o.prototype.initModels = function (e, t) {
				var i = this.props.language,
					a = h.createModel(t, i),
					s = h.createModel(e, i);
				this.editor.setModel({ original: a, modified: s });
			}),
			(o.prototype.initMonaco = function () {
				var e =
						this.props.value != null
							? this.props.value
							: this.props.defaultValue,
					t = this.props,
					i = t.original,
					a = t.theme,
					s = t.options,
					l = t.overrideServices,
					u = t.className;
				this.containerElement &&
					(this.editorWillMount(),
					(this.editor = h.createDiffEditor(
						this.containerElement,
						g(
							g(g({}, u ? { extraEditorClassName: u } : {}), s),
							a ? { theme: a } : {},
						),
						l,
					)),
					this.initModels(e, i),
					this.editorDidMount(this.editor));
			}),
			(o.prototype.destroyMonaco = function () {
				if (this.editor) {
					this.editorWillUnmount(this.editor), this.editor.dispose();
					var e = this.editor.getModel(),
						t = e.original,
						i = e.modified;
					t && t.dispose(), i && i.dispose();
				}
				this._subscription && this._subscription.dispose();
			}),
			(o.prototype.render = function () {
				var e = this.props,
					t = e.width,
					i = e.height,
					a = M(t),
					s = M(i),
					l = { width: a, height: s };
				return y.createElement("div", {
					ref: this.assignRef,
					style: l,
					className: "react-monaco-editor-container",
				});
			}),
			(o.propTypes = {
				width: n.oneOfType([n.string, n.number]),
				height: n.oneOfType([n.string, n.number]),
				original: n.string,
				value: n.string,
				defaultValue: n.string,
				language: n.string,
				theme: n.string,
				options: n.object,
				overrideServices: n.object,
				editorWillMount: n.func,
				editorDidMount: n.func,
				editorWillUnmount: n.func,
				onChange: n.func,
				className: n.string,
			}),
			(o.defaultProps = {
				width: "100%",
				height: "100%",
				original: null,
				value: null,
				defaultValue: "",
				language: "javascript",
				theme: null,
				options: {},
				overrideServices: {},
				editorWillMount: f,
				editorDidMount: f,
				editorWillUnmount: f,
				onChange: f,
				className: null,
			}),
			o
		);
	})(y.Component),
	E = (() => {
		var r = (o, e) => (
			(r =
				Object.setPrototypeOf ||
				({ __proto__: [] } instanceof Array &&
					((t, i) => {
						t.__proto__ = i;
					})) ||
				((t, i) => {
					for (var a in i) Object.hasOwn(i, a) && (t[a] = i[a]);
				})),
			r(o, e)
		);
		return (o, e) => {
			if (typeof e != "function" && e !== null)
				throw new TypeError(
					"Class extends value " + String(e) + " is not a constructor or null",
				);
			r(o, e);
			function t() {
				this.constructor = o;
			}
			o.prototype =
				e === null ? Object.create(e) : ((t.prototype = e.prototype), new t());
		};
	})(),
	p = function () {
		return (
			(p =
				Object.assign ||
				function (r) {
					for (var o, e = 1, t = arguments.length; e < t; e++) {
						o = arguments[e];
						for (var i in o) Object.hasOwn(o, i) && (r[i] = o[i]);
					}
					return r;
				}),
			p.apply(this, arguments)
		);
	},
	W = (r, o) => {
		var e = {};
		for (var t in r) Object.hasOwn(r, t) && o.indexOf(t) < 0 && (e[t] = r[t]);
		if (r != null && typeof Object.getOwnPropertySymbols == "function")
			for (var i = 0, t = Object.getOwnPropertySymbols(r); i < t.length; i++)
				o.indexOf(t[i]) < 0 &&
					Object.prototype.propertyIsEnumerable.call(r, t[i]) &&
					(e[t[i]] = r[t[i]]);
		return e;
	},
	D = ((r) => {
		E(o, r);
		function o(e) {
			var t = r.call(this, e) || this;
			return (
				(t.assignRef = (i) => {
					t.containerElement = i;
				}),
				(t.containerElement = void 0),
				t
			);
		}
		return (
			(o.prototype.componentDidMount = function () {
				this.initMonaco();
			}),
			(o.prototype.componentDidUpdate = function (e) {
				var t = this.props,
					i = t.value,
					a = t.language,
					s = t.theme,
					l = t.height,
					u = t.options,
					_ = t.width,
					v = t.className,
					c = this.editor,
					d = c.getModel();
				if (
					(this.props.value != null &&
						this.props.value !== d.getValue() &&
						((this.__prevent_trigger_change_event = !0),
						this.editor.pushUndoStop(),
						d.pushEditOperations(
							[],
							[{ range: d.getFullModelRange(), text: i }],
						),
						this.editor.pushUndoStop(),
						(this.__prevent_trigger_change_event = !1)),
					e.language !== a && h.setModelLanguage(d, a),
					e.theme !== s && h.setTheme(s),
					c && (_ !== e.width || l !== e.height) && c.layout(),
					e.options !== u)
				) {
					u.model;
					var O = W(u, ["model"]);
					c.updateOptions(p(p({}, v ? { extraEditorClassName: v } : {}), O));
				}
			}),
			(o.prototype.componentWillUnmount = function () {
				this.destroyMonaco();
			}),
			(o.prototype.destroyMonaco = function () {
				if (this.editor) {
					this.editorWillUnmount(this.editor), this.editor.dispose();
					var e = this.editor.getModel();
					e && e.dispose();
				}
				this._subscription && this._subscription.dispose();
			}),
			(o.prototype.initMonaco = function () {
				var e =
						this.props.value != null
							? this.props.value
							: this.props.defaultValue,
					t = this.props,
					i = t.language,
					a = t.theme,
					s = t.overrideServices,
					l = t.className;
				if (this.containerElement) {
					var u = p(p({}, this.props.options), this.editorWillMount());
					(this.editor = h.create(
						this.containerElement,
						p(
							p(
								p(
									{ value: e, language: i },
									l ? { extraEditorClassName: l } : {},
								),
								u,
							),
							a ? { theme: a } : {},
						),
						s,
					)),
						this.editorDidMount(this.editor);
				}
			}),
			(o.prototype.editorWillMount = function () {
				var e = this.props.editorWillMount,
					t = e(m);
				return t || {};
			}),
			(o.prototype.editorDidMount = function (e) {
				this.props.editorDidMount(e, m),
					(this._subscription = e.onDidChangeModelContent((i) => {
						this.__prevent_trigger_change_event ||
							this.props.onChange(e.getValue(), i);
					}));
			}),
			(o.prototype.editorWillUnmount = function (e) {
				var t = this.props.editorWillUnmount;
				t(e, m);
			}),
			(o.prototype.render = function () {
				var e = this.props,
					t = e.width,
					i = e.height,
					a = M(t),
					s = M(i),
					l = { width: a, height: s };
				return y.createElement("div", {
					ref: this.assignRef,
					style: l,
					className: "react-monaco-editor-container",
				});
			}),
			(o.propTypes = {
				width: n.oneOfType([n.string, n.number]),
				height: n.oneOfType([n.string, n.number]),
				value: n.string,
				defaultValue: n.string,
				language: n.string,
				theme: n.string,
				options: n.object,
				overrideServices: n.object,
				editorWillMount: n.func,
				editorDidMount: n.func,
				editorWillUnmount: n.func,
				onChange: n.func,
				className: n.string,
			}),
			(o.defaultProps = {
				width: "100%",
				height: "100%",
				value: null,
				defaultValue: "",
				language: "javascript",
				theme: null,
				options: {},
				overrideServices: {},
				editorWillMount: f,
				editorDidMount: f,
				editorWillUnmount: f,
				onChange: f,
				className: null,
			}),
			o
		);
	})(y.Component);
export { D as default, m as monaco, U as MonacoDiffEditor };

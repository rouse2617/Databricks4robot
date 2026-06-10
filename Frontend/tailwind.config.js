/** @type {import('tailwindcss').Config} */
export default {
	content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
	theme: {
		extend: {
			colors: {
				primary: "#2563eb",
				"primary-hover": "#1d4ed8",
				"primary-light": "#e8f0fe",
				success: "#16a34a",
				"success-bg": "#e6f7ef",
				warning: "#d97706",
				"warning-bg": "#fff3e0",
				error: "#dc2626",
				"error-bg": "#fdeae5",
				"text-secondary": "#64748b",
				surface: "#f8fafc",
				border: "#e2e8f0",
			},
		},
	},
	plugins: [],
	corePlugins: {
		preflight: false, // avoid conflict with Ant Design reset
	},
};

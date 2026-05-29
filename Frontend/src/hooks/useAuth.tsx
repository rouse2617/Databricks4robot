import {
	createContext,
	type ReactNode,
	useContext,
	useEffect,
	useState,
} from "react";
import { authApi } from "../api/auth";
import { UNAUTHORIZED_EVENT } from "../api/client";

const DEV_ACCESS_TOKEN = import.meta.env.DEV
	? (import.meta.env.VITE_DEV_ACCESS_TOKEN ?? "").trim()
	: "";

interface AuthContextValue {
	isAuthenticated: boolean;
	loading: boolean;
	user: { email: string; role: string } | null;
	login: (email: string) => Promise<void>;
	logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
	const [isAuthenticated, setIsAuthenticated] = useState(false);
	const [loading, setLoading] = useState(true);
	const [user, setUser] = useState<{ email: string; role: string } | null>(null);

	useEffect(() => {
		let cancelled = false;
		const handleUnauthorized = () => {
			setIsAuthenticated(false);
			setUser(null);
		};
		window.addEventListener(UNAUTHORIZED_EVENT, handleUnauthorized);

		authApi
			.me()
			.then((data) => {
				if (!cancelled) {
					setIsAuthenticated(true);
					if (data.email) setUser({ email: data.email, role: data.role ?? "user" });
				}
			})
			.catch(async () => {
				if (!DEV_ACCESS_TOKEN) {
					if (!cancelled) setIsAuthenticated(false);
					return;
				}
				try {
					await authApi.login(DEV_ACCESS_TOKEN);
					if (!cancelled) setIsAuthenticated(true);
				} catch {
					if (!cancelled) setIsAuthenticated(false);
				}
			})
			.finally(() => {
				if (!cancelled) setLoading(false);
			});

		return () => {
			cancelled = true;
			window.removeEventListener(UNAUTHORIZED_EVENT, handleUnauthorized);
		};
	}, []);

	const login = async (email: string) => {
		const data = await authApi.emailLogin(email);
		setUser({ email: data.email, role: data.role });
		setIsAuthenticated(true);
	};

	const logout = async () => {
		await authApi.logout();
		setIsAuthenticated(false);
		setUser(null);
	};

	return (
		<AuthContext.Provider value={{ isAuthenticated, loading, user, login, logout }}>
			{children}
		</AuthContext.Provider>
	);
}

export function useAuth() {
	const context = useContext(AuthContext);
	if (!context) {
		throw new Error("useAuth must be used within an AuthProvider");
	}
	return context;
}

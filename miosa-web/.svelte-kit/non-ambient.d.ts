
// this file is generated — do not edit it


declare module "svelte/elements" {
	export interface HTMLAttributes<T> {
		'data-sveltekit-keepfocus'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-noscroll'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-preload-code'?:
			| true
			| ''
			| 'eager'
			| 'viewport'
			| 'hover'
			| 'tap'
			| 'off'
			| undefined
			| null;
		'data-sveltekit-preload-data'?: true | '' | 'hover' | 'tap' | 'off' | undefined | null;
		'data-sveltekit-reload'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-replacestate'?: true | '' | 'off' | undefined | null;
	}
}

export {};


declare module "$app/types" {
	export interface AppTypes {
		RouteId(): "/" | "/analyze" | "/api" | "/api/proxy" | "/api/proxy/[...path]" | "/api/ws" | "/auth" | "/auth/login" | "/auth/signup" | "/build" | "/build/schema" | "/chat" | "/deploy" | "/expand" | "/ide" | "/onboarding" | "/optimize" | "/workspace" | "/workspace/[id]" | "/workspace/[id]/settings";
		RouteParams(): {
			"/api/proxy/[...path]": { path: string };
			"/workspace/[id]": { id: string };
			"/workspace/[id]/settings": { id: string }
		};
		LayoutParams(): {
			"/": { path?: string; id?: string };
			"/analyze": Record<string, never>;
			"/api": { path?: string };
			"/api/proxy": { path?: string };
			"/api/proxy/[...path]": { path: string };
			"/api/ws": Record<string, never>;
			"/auth": Record<string, never>;
			"/auth/login": Record<string, never>;
			"/auth/signup": Record<string, never>;
			"/build": Record<string, never>;
			"/build/schema": Record<string, never>;
			"/chat": Record<string, never>;
			"/deploy": Record<string, never>;
			"/expand": Record<string, never>;
			"/ide": Record<string, never>;
			"/onboarding": Record<string, never>;
			"/optimize": Record<string, never>;
			"/workspace": { id?: string };
			"/workspace/[id]": { id: string };
			"/workspace/[id]/settings": { id: string }
		};
		Pathname(): "/" | "/analyze" | "/api" | "/api/proxy" | `/api/proxy/${string}` & {} | "/api/ws" | "/auth" | "/auth/login" | "/auth/signup" | "/build" | "/build/schema" | "/chat" | "/deploy" | "/expand" | "/ide" | "/onboarding" | "/optimize" | "/workspace" | `/workspace/${string}` & {} | `/workspace/${string}/settings` & {};
		ResolvedPathname(): `${"" | `/${string}`}${ReturnType<AppTypes['Pathname']>}`;
		Asset(): "/favicon.ico" | "/manifest.json" | "/robots.txt" | "/service-worker.js";
	}
}
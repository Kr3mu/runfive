/** Theme state management. Persists to localStorage, applies `.dark` class on `<html>`. */

type Theme = "light" | "dark";

const STORAGE_KEY = "runfive-theme";

function getInitialTheme(): Theme {
    if (typeof window === "undefined") return "dark";
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "light" || stored === "dark") return stored;
    return window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light";
}

function applyTheme(value: Theme): void {
    document.documentElement.classList.toggle("dark", value === "dark");
}

class ThemeState {
    current = $state<Theme>(getInitialTheme());

    constructor() {
        applyTheme(this.current);
    }

    get value(): Theme {
        return this.current;
    }

    toggle(): void {
        this.current = this.current === "dark" ? "light" : "dark";
        localStorage.setItem(STORAGE_KEY, this.current);
        applyTheme(this.current);
    }
}

export const theme = new ThemeState();

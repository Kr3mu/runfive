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

function applyTheme(value: Theme, animate: boolean = false): void {
    if (animate) {
        const toDark = value === "dark";
        const overlay = document.createElement("div");
        overlay.style.cssText = `
            position: fixed; inset: 0; z-index: 9999;
            background: ${toDark ? "#0a0a0b" : "#f0ede6"};
            transform: translateX(${toDark ? "-100%" : "100%"});
            transition: transform 0.4s cubic-bezier(0.4, 0, 0.2, 1);
            pointer-events: none;
        `;
        document.body.appendChild(overlay);

        overlay.getBoundingClientRect();  // Required by firefox - to force reflow

        requestAnimationFrame((): void => {
            overlay.style.transform = "translateX(0)";

            overlay.addEventListener("transitionend", (): void => {
                document.documentElement.classList.toggle("dark", toDark);

                requestAnimationFrame((): void => {
                    overlay.style.transform = `translateX(${toDark ? "100%" : "-100%"})`;
                    overlay.addEventListener("transitionend", (): void => {
                        overlay.remove();
                    }, { once: true });
                });
            }, { once: true });
        });
    } else {
        document.documentElement.classList.toggle("dark", value === "dark");
    }
}

let current = $state<Theme>(getInitialTheme());
applyTheme(getInitialTheme());

export const theme = {
    get value(): Theme {
        return current;
    },
    toggle(): void {
        current = current === "dark" ? "light" : "dark";
        localStorage.setItem(STORAGE_KEY, current);
        applyTheme(current, true);
    },
};

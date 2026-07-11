type Theme = 'light' | 'dark';

const THEME_KEY = 'theme';

let previousTheme: Theme = 'light';

function getThemeFromStorage(): Theme | null {
    return localStorage.getItem(THEME_KEY) as Theme | null;
}

function getSystemPreference(): Theme {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function updateToggleIcons(theme: Theme): void {
    const themeIcon = document.querySelector('#theme-toggle i');
    if (themeIcon) {
        themeIcon.className = theme === 'dark' ? 'fa-solid fa-sun' : 'fa-solid fa-moon';
    }
}

function setTheme(theme: Theme): void {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem(THEME_KEY, theme);
    updateToggleIcons(theme);
}

function initTheme(): void {
    let theme: Theme;

    const stored = getThemeFromStorage();
    if (stored && (stored === 'light' || stored === 'dark')) {
        theme = stored;
    } else {
        theme = getSystemPreference();
    }

    previousTheme = theme;
    setTheme(theme);
}

function cycleTheme(): void {
    const current = (document.documentElement.getAttribute('data-theme') || 'light') as Theme;
    const next = current === 'light' ? 'dark' : 'light';
    previousTheme = next;
    setTheme(next);
}

function getTheme(): Theme {
    return (document.documentElement.getAttribute('data-theme') || 'light') as Theme;
}

initTheme();

const themeToggle = document.getElementById('theme-toggle');
if (themeToggle) {
    themeToggle.addEventListener('click', cycleTheme);
}

export { initTheme, setTheme, cycleTheme, getTheme };
export type { Theme };

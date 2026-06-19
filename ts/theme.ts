type Theme = 'light' | 'dark' | 'eink';

const THEME_KEY = 'theme';

let previousTheme: Theme = 'light';

function getThemeFromUrl(): Theme | null {
    const params = new URLSearchParams(window.location.search);
    if (params.get('eink') === '1') return 'eink';
    if (params.get('eink') === '0') return null;
    return null;
}

function getThemeFromStorage(): Theme | null {
    return localStorage.getItem(THEME_KEY) as Theme | null;
}

function getSystemPreference(): Theme {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function isEinkAdminForced(): boolean {
    return document.documentElement.getAttribute('data-eink') === '1'
        || document.body.getAttribute('data-eink') === '1';
}

function updateToggleIcons(theme: Theme): void {
    const themeIcon = document.querySelector('#theme-toggle i');
    if (themeIcon) {
        if (theme === 'eink') {
            themeIcon.className = 'fa-solid fa-moon';
        } else {
            themeIcon.className = theme === 'dark' ? 'fa-solid fa-sun' : 'fa-solid fa-moon';
        }
    }

    const einkToggle = document.getElementById('eink-toggle');
    if (einkToggle) {
        const icon = einkToggle.querySelector('i');
        const isEink = theme === 'eink';
        if (icon) {
            icon.className = isEink ? 'fa-solid fa-file-lines' : 'fa-solid fa-file-pen';
        }
        einkToggle.setAttribute('aria-label', isEink ? 'Disable E-Ink Mode' : 'Enable E-Ink Mode');
    }
}

function setTheme(theme: Theme): void {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem(THEME_KEY, theme);
    updateToggleIcons(theme);
}

function initTheme(): void {
    let theme: Theme;

    if (isEinkAdminForced()) {
        theme = 'eink';
    } else {
        const urlTheme = getThemeFromUrl();
        if (urlTheme === 'eink') {
            theme = 'eink';
        } else {
            const stored = getThemeFromStorage();
            if (stored && (stored === 'light' || stored === 'dark' || stored === 'eink')) {
                theme = stored;
            } else {
                theme = getSystemPreference();
            }
        }
    }

    if (theme !== 'eink') {
        previousTheme = theme;
    }

    setTheme(theme);
}

function cycleTheme(): void {
    const current = (document.documentElement.getAttribute('data-theme') || 'light') as Theme;
    if (current === 'eink') {
        setTheme(previousTheme);
    } else {
        const next = current === 'light' ? 'dark' : 'light';
        previousTheme = next;
        setTheme(next);
    }
}

function toggleEink(): void {
    const current = document.documentElement.getAttribute('data-theme') || 'light';
    if (current === 'eink') {
        setTheme(previousTheme);
    } else {
        previousTheme = current as Theme;
        setTheme('eink');
    }
}

function getTheme(): Theme {
    return (document.documentElement.getAttribute('data-theme') || 'light') as Theme;
}

initTheme();

const themeToggle = document.getElementById('theme-toggle');
if (themeToggle) {
    themeToggle.addEventListener('click', cycleTheme);
}

const einkToggle = document.getElementById('eink-toggle');
if (einkToggle) {
    einkToggle.addEventListener('click', toggleEink);
}

export { initTheme, setTheme, cycleTheme, toggleEink, getTheme };
export type { Theme };

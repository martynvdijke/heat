interface Translations {
  [key: string]: string;
}

interface I18nModule {
  t(key: string): string;
  load(lang?: string): Promise<void>;
  apply(): void;
  switchLang(lang: string): Promise<void>;
  getCurrentLang(): string;
}

declare var i18n: I18nModule;

const i18nModule = ((): I18nModule => {
  let translations: Translations = {};
  let currentLang = 'en';

  function load(lang?: string): Promise<void> {
    let url = '/api/translations';
    if (lang) url += '?lang=' + lang;
    return fetch(url)
      .then(r => r.json())
      .then((t: Translations) => {
        translations = t;
        currentLang = t._lang || 'en';
        apply();
      })
      .catch(() => {});
  }

  function t(key: string): string {
    return translations[key] || key;
  }

  function apply(): void {
    document.querySelectorAll('[data-i18n]').forEach(el => {
      el.textContent = t(el.getAttribute('data-i18n')!);
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
      (el as HTMLInputElement).placeholder = t(el.getAttribute('data-i18n-placeholder')!);
    });
    document.querySelectorAll('[data-i18n-title]').forEach(el => {
      el.setAttribute('title', t(el.getAttribute('data-i18n-title')!));
    });
  }

  function switchLang(lang: string): Promise<void> {
    return fetch('/api/language', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ lang: lang }),
    }).then(() => load(lang));
  }

  function getCurrentLang(): string {
    return currentLang;
  }

  load();

  return { t, load, apply, switchLang, getCurrentLang };
})();

var i18n: I18nModule = i18nModule;

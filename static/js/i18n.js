"use strict";
const i18nModule = (() => {
    let translations = {};
    let currentLang = 'en';
    function load(lang) {
        let url = '/api/translations';
        if (lang)
            url += '?lang=' + lang;
        return fetch(url)
            .then(r => r.json())
            .then((t) => {
            translations = t;
            currentLang = t._lang || 'en';
            apply();
        })
            .catch(() => { });
    }
    function t(key) {
        return translations[key] || key;
    }
    function apply() {
        document.querySelectorAll('[data-i18n]').forEach(el => {
            el.textContent = t(el.getAttribute('data-i18n'));
        });
        document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            el.placeholder = t(el.getAttribute('data-i18n-placeholder'));
        });
        document.querySelectorAll('[data-i18n-title]').forEach(el => {
            el.setAttribute('title', t(el.getAttribute('data-i18n-title')));
        });
    }
    function switchLang(lang) {
        return fetch('/api/language', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ lang: lang }),
        }).then(() => load(lang));
    }
    function getCurrentLang() {
        return currentLang;
    }
    load();
    return { t, load, apply, switchLang, getCurrentLang };
})();
var i18n = i18nModule;
//# sourceMappingURL=i18n.js.map
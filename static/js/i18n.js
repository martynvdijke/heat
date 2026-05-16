var i18n = (function() {
    var translations = {};
    var currentLang = 'en';

    function load(lang) {
        var url = '/api/translations';
        if (lang) url += '?lang=' + lang;
        return fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(t) {
                translations = t;
                currentLang = t._lang || 'en';
                apply();
            })
            .catch(function() {});
    }

    function t(key) {
        return translations[key] || key;
    }

    function apply() {
        document.querySelectorAll('[data-i18n]').forEach(function(el) {
            el.textContent = t(el.getAttribute('data-i18n'));
        });
        document.querySelectorAll('[data-i18n-placeholder]').forEach(function(el) {
            el.placeholder = t(el.getAttribute('data-i18n-placeholder'));
        });
        document.querySelectorAll('[data-i18n-title]').forEach(function(el) {
            el.title = t(el.getAttribute('data-i18n-title'));
        });
    }

    function switchLang(lang) {
        return fetch('/api/language', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({lang: lang})
        }).then(function() {
            return load(lang);
        });
    }

    function getCurrentLang() {
        return currentLang;
    }

    load();

    return { t: t, load: load, apply: apply, switchLang: switchLang, getCurrentLang: getCurrentLang };
})();

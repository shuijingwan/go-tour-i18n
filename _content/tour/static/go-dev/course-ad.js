// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
(function(window, document) {
    'use strict';

    function report(error) {
        if (window.console && typeof window.console.warn === 'function') {
            window.console.warn('course ad request failed', error);
        }
    }

    function mount(element) {
        if (element.querySelector('ins.adsbygoogle')) {
            return;
        }

        element.setAttribute('role', 'complementary');
        element.setAttribute('aria-label', 'Advertisement');

        var ad = document.createElement('ins');
        ad.className = 'adsbygoogle';
        ad.style.display = 'block';
        ad.setAttribute('data-ad-client', 'ca-pub-8392190980622725');
        ad.setAttribute('data-ad-slot', '4728596962');
        ad.setAttribute('data-ad-format', 'auto');
        ad.setAttribute('data-full-width-responsive', 'true');
        element.appendChild(ad);

        try {
            (window.adsbygoogle = window.adsbygoogle || []).push({});
        } catch (error) {
            report(error);
        }
    }

    function unmount(element) {
        var ads = element.querySelectorAll('ins.adsbygoogle');
        for (var i = 0; i < ads.length; i++) {
            ads[i].parentNode.removeChild(ads[i]);
        }
        element.removeAttribute('role');
        element.removeAttribute('aria-label');
    }

    // Angular owns when a course view is linked and destroyed. This helper
    // deliberately does not observe the document or make route choices.
    window.goDevCourseAd = {
        mount: mount,
        unmount: unmount
    };
}(window, document));

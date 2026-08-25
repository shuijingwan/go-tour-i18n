// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
(function() {
    'use strict';

    var selector = '[data-go-dev-course-ad]';
    var mountedAttribute = 'data-go-dev-course-ad-mounted';

    function mount(element) {
        if (element.hasAttribute(mountedAttribute)) {
            return;
        }
        element.setAttribute(mountedAttribute, 'true');
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

        (window.adsbygoogle = window.adsbygoogle || []).push({});
    }

    function mountWithin(root) {
        if (root.nodeType !== 1) {
            return;
        }
        if (root.matches(selector)) {
            mount(root);
        }
        var elements = root.querySelectorAll(selector);
        for (var i = 0; i < elements.length; i++) {
            mount(elements[i]);
        }
    }

    function mountAll() {
        var elements = document.querySelectorAll(selector);
        for (var i = 0; i < elements.length; i++) {
            mount(elements[i]);
        }
    }

    mountAll();

    var observer = new MutationObserver(function(mutations) {
        for (var i = 0; i < mutations.length; i++) {
            for (var j = 0; j < mutations[i].addedNodes.length; j++) {
                mountWithin(mutations[i].addedNodes[j]);
            }
        }
    });
    observer.observe(document.body, {childList: true, subtree: true});
}());

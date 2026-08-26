// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
(function(window, document) {
    'use strict';

    var layoutProtectionKey = '__goDevCourseAdLayoutProtection';

    function report(error) {
        if (window.console && typeof window.console.warn === 'function') {
            window.console.warn('course ad request failed', error);
        }
    }

    function directChild(element, selector) {
        for (var i = 0; i < element.children.length; i++) {
            if (element.children[i].matches(selector)) {
                return element.children[i];
            }
        }
        return null;
    }

    // Funding Choices and AdSense have been observed applying these exact
    // declarations to the course editor's ancestors. Do not remove any other
    // inline styles: the editor and AdSense both legitimately use them.
    function removeAdSenseHeightOverrides(element) {
        if (element.style.getPropertyValue('height') === 'auto' &&
                element.style.getPropertyPriority('height') === 'important') {
            element.style.removeProperty('height');
        }
        if (element.style.getPropertyValue('min-height') === '0px' &&
                element.style.getPropertyPriority('min-height') === 'important') {
            element.style.removeProperty('min-height');
        }
    }

    function stopProtectingLayout(element) {
        var protection = element[layoutProtectionKey];
        if (!protection) {
            return;
        }
        protection.observer.disconnect();
        delete element[layoutProtectionKey];
    }

    function protectLayoutForMount(element) {
        if (element[layoutProtectionKey]) {
            return;
        }

        var editorContainer = element.closest('#editor-container');
        var editorContent = editorContainer && directChild(editorContainer, '.relative-content');
        var leftSide = editorContent && directChild(editorContent, '#left-side');
        var leftContent = leftSide && directChild(leftSide, '.relative-content');
        if (!leftContent) {
            return;
        }

        var nodes = [editorContainer, editorContent, leftSide, leftContent];
        for (var i = 0; i < nodes.length; i++) {
            removeAdSenseHeightOverrides(nodes[i]);
        }

        var observer = new MutationObserver(function(mutations) {
            for (var j = 0; j < mutations.length; j++) {
                removeAdSenseHeightOverrides(mutations[j].target);
            }
        });
        for (var k = 0; k < nodes.length; k++) {
            observer.observe(nodes[k], {
                attributes: true,
                attributeFilter: ['style']
            });
        }
        element[layoutProtectionKey] = {observer: observer};
    }

    function mount(element) {
        protectLayoutForMount(element);
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
        stopProtectingLayout(element);
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

// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
(function(window, document) {
    'use strict';

    var layoutProtectionKey = '__goDevCourseAdLayoutProtection';
    var experimentStorageKey = 'goDevCourseAdExperimentGroup';
    var experimentWindowKey = '__goDevCourseAdExperimentGroup';
    var experimentGroups = [
        {name: 'A', slot: '3362554728', maxWidth: 336},
        {name: 'B', slot: '1260537939', maxWidth: 468},
        {name: 'C', slot: '4220340824', maxWidth: 728},
        {name: 'D', slot: '4728596962'}
    ];

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

    function experimentGroupByName(name) {
        for (var i = 0; i < experimentGroups.length; i++) {
            if (experimentGroups[i].name === name) {
                return experimentGroups[i];
            }
        }
        return null;
    }

    // Keep a visitor in one group for the browser tab's session. sessionStorage
    // also survives a reload; the window property preserves SPA stability when
    // storage is unavailable (for example, because privacy settings block it).
    function selectExperimentGroup() {
        var group = experimentGroupByName(window[experimentWindowKey]);
        if (group) {
            return group;
        }

        try {
            group = experimentGroupByName(window.sessionStorage.getItem(experimentStorageKey));
            if (group) {
                window[experimentWindowKey] = group.name;
                return group;
            }
        } catch (error) {
            // The in-memory fallback below is sufficient for this SPA session.
        }

        group = experimentGroups[Math.floor(Math.random() * experimentGroups.length)];
        window[experimentWindowKey] = group.name;
        try {
            window.sessionStorage.setItem(experimentStorageKey, group.name);
        } catch (error) {
            // Keep the selected group in this window when storage is unavailable.
        }
        return group;
    }

    function applyExperimentGroup(element, group) {
        element.setAttribute('data-go-dev-course-ad-group', group.name);
        if (group.maxWidth) {
            element.classList.add('go-dev-course-ad--max-' + group.maxWidth);
        }
    }

    function removeExperimentGroup(element) {
        for (var i = 0; i < experimentGroups.length; i++) {
            if (experimentGroups[i].maxWidth) {
                element.classList.remove('go-dev-course-ad--max-' + experimentGroups[i].maxWidth);
            }
        }
        element.removeAttribute('data-go-dev-course-ad-group');
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

        var group = selectExperimentGroup();
        applyExperimentGroup(element, group);

        element.setAttribute('role', 'complementary');
        element.setAttribute('aria-label', 'Advertisement');

        var ad = document.createElement('ins');
        ad.className = 'adsbygoogle';
        ad.style.display = 'block';
        ad.setAttribute('data-ad-client', 'ca-pub-8392190980622725');
        ad.setAttribute('data-ad-slot', group.slot);
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
        removeExperimentGroup(element);
    }

    // Angular owns when a course view is linked and destroyed. This helper
    // deliberately does not observe the document or make route choices.
    window.goDevCourseAd = {
        mount: mount,
        unmount: unmount
    };
}(window, document));

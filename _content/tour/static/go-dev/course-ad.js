// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
(function() {
    'use strict';

    var selector = '[data-go-dev-course-ad]';
    var mountedAttribute = 'data-go-dev-course-ad-mounted';
    var layoutObserver = null;
    var currentMount = null;

    function directChild(element, childSelector) {
        for (var i = 0; i < element.children.length; i++) {
            if (element.children[i].matches(childSelector)) {
                return element.children[i];
            }
        }
        return null;
    }

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

    function stopProtectingLayout() {
        if (layoutObserver) {
            layoutObserver.disconnect();
        }
        layoutObserver = null;
        currentMount = null;
    }

    function protectLayoutForMount(element) {
        var editorContainer = element.closest('#editor-container');
        if (!editorContainer) {
            return;
        }
        var editorContent = directChild(editorContainer, '.relative-content');
        var leftSide = editorContent && directChild(editorContent, '#left-side');
        var leftContent = leftSide && directChild(leftSide, '.relative-content');
        if (!editorContent || !leftSide || !leftContent) {
            return;
        }

        var nodes = [editorContainer, editorContent, leftSide, leftContent];
        if (currentMount === element) {
            return;
        }
        stopProtectingLayout();
        currentMount = element;
        for (var i = 0; i < nodes.length; i++) {
            removeAdSenseHeightOverrides(nodes[i]);
        }

        layoutObserver = new MutationObserver(function(mutations) {
            for (var i = 0; i < mutations.length; i++) {
                removeAdSenseHeightOverrides(mutations[i].target);
            }
        });
        for (var j = 0; j < nodes.length; j++) {
            layoutObserver.observe(nodes[j], {
                attributes: true,
                attributeFilter: ['style']
            });
        }
    }

    function mount(element) {
        protectLayoutForMount(element);
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
        if (currentMount && !document.documentElement.contains(currentMount)) {
            stopProtectingLayout();
        }
    });
    observer.observe(document.body, {childList: true, subtree: true});
}());

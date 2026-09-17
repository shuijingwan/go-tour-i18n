// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

(function() {
  'use strict';

  function fallbackCopy(value) {
    var input = document.createElement('textarea');
    input.value = value;
    input.setAttribute('readonly', '');
    input.style.position = 'fixed';
    input.style.opacity = '0';
    document.body.appendChild(input);
    input.select();
    var copied = document.execCommand('copy');
    document.body.removeChild(input);
    if (!copied) throw new Error('copy command failed');
  }

  function copyText(value) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(value).catch(function() {
        fallbackCopy(value);
      });
    }
    return new Promise(function(resolve, reject) {
      try {
        fallbackCopy(value);
        resolve();
      } catch (error) {
        reject(error);
      }
    });
  }

  var toast = document.querySelector('[data-copy-toast]');
  var toastTimer;

  function syncToastToVisualViewport() {
    if (!toast || !window.visualViewport) return;
    var viewport = window.visualViewport;
    toast.style.setProperty('--support-copy-toast-center-x', viewport.offsetLeft + viewport.width / 2 + 'px');
    toast.style.setProperty('--support-copy-toast-center-y', viewport.offsetTop + viewport.height / 2 + 'px');
    toast.style.setProperty('--support-copy-toast-viewport-width', viewport.width + 'px');
  }

  if (toast && window.visualViewport) {
    syncToastToVisualViewport();
    window.visualViewport.addEventListener('resize', syncToastToVisualViewport);
    window.visualViewport.addEventListener('scroll', syncToastToVisualViewport);
  }

  function hideToast() {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toastTimer = undefined;
    toast.hidden = true;
    toast.textContent = '';
  }

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    syncToastToVisualViewport();
    toast.textContent = message;
    toast.hidden = false;
    toastTimer = window.setTimeout(hideToast, 1600);
  }

  document.addEventListener('click', function(event) {
    var button = event.target.closest && event.target.closest('[data-copy-value]');
    if (!button) return;
    var success = button.getAttribute('data-copy-success');
    hideToast();
    copyText(button.getAttribute('data-copy-value')).then(function() {
      showToast(success);
    }).catch(function() {});
  });
}());

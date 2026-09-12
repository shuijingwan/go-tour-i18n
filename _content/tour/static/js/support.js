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

  document.addEventListener('click', function(event) {
    var button = event.target.closest && event.target.closest('[data-copy-value]');
    if (!button) return;
    var label = button.getAttribute('data-copy-label');
    var success = button.getAttribute('data-copy-success');
    copyText(button.getAttribute('data-copy-value')).then(function() {
      button.textContent = success;
      window.clearTimeout(button.copyResetTimer);
      button.copyResetTimer = window.setTimeout(function() {
        button.textContent = label;
      }, 1600);
    }).catch(function() {
      button.textContent = label;
    });
  });
}());

// Copyright 2026 shing1211 — SPDX-License-Identifier: Apache-2.0

(function () {
  'use strict';

  var input = document.getElementById('search');
  if (!input) return;

  var symbols = document.querySelectorAll('.symbol');
  var sections = document.querySelectorAll('section');
  var noResults = document.getElementById('no-results');

  input.addEventListener('input', function () {
    var q = input.value.toLowerCase().trim();
    var visibleCount = 0;

    symbols.forEach(function (el) {
      var text = el.textContent.toLowerCase();
      var match = !q || text.indexOf(q) !== -1;
      el.classList.toggle('hidden', !match);
      if (match) visibleCount++;
    });

    sections.forEach(function (sec) {
      var vis = sec.querySelectorAll('.symbol:not(.hidden)');
      sec.style.display = vis.length === 0 ? 'none' : '';
    });

    if (noResults) {
      noResults.style.display = visibleCount === 0 ? '' : 'none';
    }
  });
})();

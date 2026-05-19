(function () {
  'use strict';

  var content = document.getElementById('admin-content');
  var navLinks = document.querySelectorAll('.admin-nav-link');

  function getPageFromPath() {
    var path = window.location.pathname;
    var match = path.match(/\/admin\/([^/?#]*)/);
    return match ? match[1] : 'dashboard';
  }

  function setActiveNav(page) {
    navLinks.forEach(function (link) {
      link.classList.toggle('active', link.getAttribute('data-page') === page);
    });
  }

  function renderDashboard() {
    var base = getAdminBase();
    var dashboardPath = base.replace(/\/admin$/, '') + '/dashboard';
    content.innerHTML =
      '<h2 class="admin-page-title">Dashboard</h2>' +
      '<div class="admin-card">' +
      '<p class="admin-placeholder">Metrics &amp; health (P2: embed dashboard here).</p>' +
      '<p class="admin-placeholder" style="margin-top:8px">Legacy: <a href="' + dashboardPath + '" target="_blank" rel="noopener" style="color:#38bdf8">Dashboard</a></p>' +
      '</div>';
  }

  function renderStorage() {
    content.innerHTML =
      '<h2 class="admin-page-title">Storage</h2>' +
      '<div class="admin-card">' +
      '<p class="admin-placeholder">Buckey Storage – list keys, get/put/delete (P4).</p>' +
      '</div>';
  }

  function renderCI() {
    content.innerHTML =
      '<h2 class="admin-page-title">CI</h2>' +
      '<div class="admin-card">' +
      '<p class="admin-placeholder">Workflows, Run, Executions (P5).</p>' +
      '</div>';
  }

  function getAdminBase() {
    var path = window.location.pathname;
    var i = path.indexOf('/admin');
    return i >= 0 ? path.slice(0, i + 6) : '/admin';
  }

  function render(page) {
    setActiveNav(page);
    if (page === 'dashboard') renderDashboard();
    else if (page === 'storage') renderStorage();
    else if (page === 'ci') renderCI();
    else renderDashboard();
  }

  function init() {
    var page = getPageFromPath();
    render(page);

    navLinks.forEach(function (link) {
      link.addEventListener('click', function (e) {
        e.preventDefault();
        var page = link.getAttribute('data-page');
        var base = getAdminBase();
        window.history.pushState({ page: page }, '', base + '/' + page);
        render(page);
      });
    });

    window.addEventListener('popstate', function () {
      render(getPageFromPath());
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();

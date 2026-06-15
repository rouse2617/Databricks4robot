(function () {
  var defaultOperation = '#tag/auth/POST/api/v1/auth/email-login';
  var path = window.location.pathname.replace(/\/$/, '');

  if (path === '/doc/api/reference' && !window.location.hash) {
    window.location.replace(window.location.pathname + defaultOperation);
  }
})();

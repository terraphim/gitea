export function urlQueryEscape(s: string) {
  // See "TestQueryEscape" in backend
  // https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent#encoding_for_rfc3986
  return encodeURIComponent(s).replace(
    /[!'()*]/g,
    (c) => `%${c.charCodeAt(0).toString(16).toUpperCase()}`,
  ).replaceAll('%20', '+');
}

export function pathEscape(s: string): string {
  // See "TestPathEscape" in backend
  return encodeURIComponent(s).replace(
    /[!'()*]/g,
    (c) => `%${c.charCodeAt(0).toString(16).toUpperCase()}`,
  ).replaceAll(/%(\w\w)/g, (v) => {
    switch (v) {
      case '%24': return '$';
      case '%26': return '&';
      case '%2B': return '+';
      case '%3A': return ':';
      case '%3D': return '=';
      case '%40': return '@';
      default: return v;
    }
  });
}

export function pathEscapeSegments(s: string): string {
  // The same as backend's PathEscapeSegments
  return s.split('/').map(pathEscape).join('/');
}

/** Convert an absolute or relative URL to an absolute URL with the current origin. It only
 *  processes absolute HTTP/HTTPS URLs or relative URLs like '/xxx' or '//host/xxx'. */
export function toOriginUrl(urlStr: string) {
  try {
    if (urlStr.startsWith('http://') || urlStr.startsWith('https://') || urlStr.startsWith('/')) {
      const {origin, protocol, hostname, port} = window.location;
      const url = new URL(urlStr, origin);
      url.protocol = protocol;
      url.hostname = hostname;
      url.port = port || (protocol === 'https:' ? '443' : '80');
      return url.toString();
    }
  } catch {}
  return urlStr;
}

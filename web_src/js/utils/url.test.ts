import {linkifyURLs, pathEscape, pathEscapeSegments, toOriginUrl, urlQueryEscape} from './url.ts';

describe('escape', () => {
  const queryNonAscii = " !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~";
  test('urlQueryEscape', () => {
    const expected = '+%21%22%23%24%25%26%27%28%29%2A%2B%2C-.%2F%3A%3B%3C%3D%3E%3F%40%5B%5C%5D%5E_%60%7B%7C%7D~';
    expect(urlQueryEscape(queryNonAscii)).toEqual(expected);
  });

  test('pathEscape', () => {
    const expected = '%20%21%22%23$%25&%27%28%29%2A+%2C-.%2F:%3B%3C=%3E%3F@%5B%5C%5D%5E_%60%7B%7C%7D~';
    expect(pathEscape(queryNonAscii)).toEqual(expected);
  });

  test('pathEscapeSegments', () => {
    expect(pathEscapeSegments('a/b/c')).toEqual('a/b/c');
    expect(pathEscapeSegments('a/b/ c')).toEqual('a/b/%20c');
    expect(pathEscapeSegments('a/b+c')).toEqual('a/b+c');
  });
});

test('toOriginUrl', () => {
  const oldLocation = String(window.location);
  for (const origin of ['https://example.com', 'https://example.com:3000']) {
    window.location.assign(`${origin}/`);
    expect(toOriginUrl('/')).toEqual(`${origin}/`);
    expect(toOriginUrl('/org/repo.git')).toEqual(`${origin}/org/repo.git`);
    expect(toOriginUrl('https://another.com')).toEqual(`${origin}/`);
    expect(toOriginUrl('https://another.com/')).toEqual(`${origin}/`);
    expect(toOriginUrl('https://another.com/org/repo.git')).toEqual(`${origin}/org/repo.git`);
    expect(toOriginUrl('https://another.com:4000')).toEqual(`${origin}/`);
    expect(toOriginUrl('https://another.com:4000/')).toEqual(`${origin}/`);
    expect(toOriginUrl('https://another.com:4000/org/repo.git')).toEqual(`${origin}/org/repo.git`);
  }
  window.location.assign(oldLocation);
});

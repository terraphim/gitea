// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package httplib

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"code.gitea.io/gitea/modules/typesniffer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeContentByReader(t *testing.T) {
	data := "0123456789abcdef"

	test := func(t *testing.T, expectedStatusCode int, expectedContent string) {
		_, rangeStr, _ := strings.Cut(t.Name(), "_range_")
		r := &http.Request{Header: http.Header{}, Form: url.Values{}}
		if rangeStr != "" {
			r.Header.Set("Range", "bytes="+rangeStr)
		}
		reader := strings.NewReader(data)
		w := httptest.NewRecorder()
		ServeContentByReader(r, w, int64(len(data)), reader, &ServeHeaderOptions{})
		assert.Equal(t, expectedStatusCode, w.Code)
		if expectedStatusCode == http.StatusPartialContent || expectedStatusCode == http.StatusOK {
			assert.Equal(t, strconv.Itoa(len(expectedContent)), w.Header().Get("Content-Length"))
			assert.Equal(t, expectedContent, w.Body.String())
			// Pick 7 (#37455) regression: every served file gets a CSP header.
			// The detected content type for plain ASCII is text/plain, which
			// falls under the default sandbox CSP (not PDF, not audio/video).
			assert.Equal(t, serveHeaderCspDefault, w.Header().Get("Content-Security-Policy"))
		}
	}

	t.Run("_range_", func(t *testing.T) {
		test(t, http.StatusOK, data)
	})
	t.Run("_range_0-", func(t *testing.T) {
		test(t, http.StatusPartialContent, data)
	})
	t.Run("_range_0-15", func(t *testing.T) {
		test(t, http.StatusPartialContent, data)
	})
	t.Run("_range_1-", func(t *testing.T) {
		test(t, http.StatusPartialContent, data[1:])
	})
	t.Run("_range_1-3", func(t *testing.T) {
		test(t, http.StatusPartialContent, data[1:3+1])
	})
	t.Run("_range_16-", func(t *testing.T) {
		test(t, http.StatusRequestedRangeNotSatisfiable, "")
	})
	t.Run("_range_1-99999", func(t *testing.T) {
		test(t, http.StatusPartialContent, data[1:])
	})
}

func TestServeSetContentSecurityHeaders(t *testing.T) {
	// Sanity-check the default policy still carries the sandbox attribute --
	// a regression here would silently weaken the SVG sandbox.
	assert.Contains(t, serveHeaderCspDefault, "; sandbox")

	cases := []struct {
		name        string
		contentType string
		expected    string // "" means: no Content-Security-Policy header
	}{
		{"empty content type uses default", "", serveHeaderCspDefault},
		{"unknown content type uses default", "any", serveHeaderCspDefault},
		{"svg uses default sandbox", typesniffer.MimeTypeImageSvg, serveHeaderCspDefault},
		{"html uses default sandbox", "text/html; charset=utf-8", serveHeaderCspDefault},
		{"pdf drops sandbox", "application/pdf", serveHeaderCspPdf},
		{"pdf with charset still drops sandbox", "application/pdf; charset=utf-8", serveHeaderCspPdf},
		{"audio is exempt", "audio/mp4", ""},
		{"audio with codecs is exempt", "audio/ogg; codecs=opus", ""},
		{"video is exempt", "video/mp4", ""},
		{"video with codecs is exempt", "video/ogg; codecs=theora,vorbis", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			// Seed with a non-empty value so the audio/video exemption can
			// also assert the header is removed (not merely absent).
			w.Header().Set("Content-Security-Policy", "should-be-replaced")

			serveSetContentSecurityHeaders(w, tc.contentType)

			if tc.expected == "" {
				_, present := w.Header()["Content-Security-Policy"]
				assert.False(t, present,
					"expected CSP header to be removed for %q, got %q",
					tc.contentType, w.Header().Get("Content-Security-Policy"))
			} else {
				assert.Equal(t, tc.expected, w.Header().Get("Content-Security-Policy"))
			}
		})
	}
}

func TestServeContentByReadSeeker(t *testing.T) {
	data := "0123456789abcdef"
	tmpFile := t.TempDir() + "/test"
	err := os.WriteFile(tmpFile, []byte(data), 0o644)
	assert.NoError(t, err)

	test := func(t *testing.T, expectedStatusCode int, expectedContent string) {
		_, rangeStr, _ := strings.Cut(t.Name(), "_range_")
		r := &http.Request{Header: http.Header{}, Form: url.Values{}}
		if rangeStr != "" {
			r.Header.Set("Range", "bytes="+rangeStr)
		}

		seekReader, err := os.OpenFile(tmpFile, os.O_RDONLY, 0o644)
		require.NoError(t, err)
		defer seekReader.Close()

		w := httptest.NewRecorder()
		ServeContentByReadSeeker(r, w, nil, seekReader, &ServeHeaderOptions{})
		assert.Equal(t, expectedStatusCode, w.Code)
		if expectedStatusCode == http.StatusPartialContent || expectedStatusCode == http.StatusOK {
			assert.Equal(t, strconv.Itoa(len(expectedContent)), w.Header().Get("Content-Length"))
			assert.Equal(t, expectedContent, w.Body.String())
			// Pick 7 (#37455) regression: ServeSetHeaders writes a CSP header
			// for default served content (the test fixture is plain ASCII).
			assert.Equal(t, serveHeaderCspDefault, w.Header().Get("Content-Security-Policy"))
		}
	}

	t.Run("_range_", func(t *testing.T) {
		test(t, http.StatusOK, data)
	})
	t.Run("_range_0-", func(t *testing.T) {
		test(t, http.StatusPartialContent, data)
	})
	t.Run("_range_0-15", func(t *testing.T) {
		test(t, http.StatusPartialContent, data)
	})
	t.Run("_range_1-", func(t *testing.T) {
		test(t, http.StatusPartialContent, data[1:])
	})
	t.Run("_range_1-3", func(t *testing.T) {
		test(t, http.StatusPartialContent, data[1:3+1])
	})
	t.Run("_range_16-", func(t *testing.T) {
		test(t, http.StatusRequestedRangeNotSatisfiable, "")
	})
	t.Run("_range_1-99999", func(t *testing.T) {
		test(t, http.StatusPartialContent, data[1:])
	})
}

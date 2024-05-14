/**
 * @Time: 2024/5/14 9:44
 * @Author: LiuKun
 * @File: utils.go
 * @Description:
 */

package cdputil

import (
	"context"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/storage"
	"net/http"
	"strings"
	"time"
)

// HeaderFromMap map[string]string转为network.Headers
func HeaderFromMap(header map[string]string) network.Headers {
	hs := make(network.Headers)
	for k, v := range header {
		hs[k] = v
	}
	return hs
}

// HeaderConverter network.Headers转为[]*fetch.HeaderEntry
func HeaderConverter(header network.Headers) []*fetch.HeaderEntry {
	hs := make([]*fetch.HeaderEntry, 0)
	for k, v := range header {
		hs = append(hs, &fetch.HeaderEntry{Name: k, Value: v.(string)})
	}
	return hs
}

// CookieConverter []*network.Cookie转为[]*http.Cookie
func CookieConverter(cookies []*network.Cookie) []*http.Cookie {
	cs := make([]*http.Cookie, 0)
	for _, cookie := range cookies {

		c := &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			HttpOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
			Expires:  time.Unix(int64(cookie.Expires), 0),
		}

		lowerVal := strings.ToLower(cookie.SameSite.String())
		switch lowerVal {
		case "lax":
			c.SameSite = http.SameSiteLaxMode
		case "strict":
			c.SameSite = http.SameSiteStrictMode
		case "none":
			c.SameSite = http.SameSiteNoneMode
		default:
			c.SameSite = http.SameSiteDefaultMode
		}
		cs = append(cs, c)
	}
	return cs
}

// GetCookies 获取特定域名的Cookies
func GetCookies(ctx context.Context, domain string) ([]*http.Cookie, error) {
	cookies, err := storage.GetCookies().Do(ctx)
	if err != nil {
		return nil, err
	}

	newCookies := make([]*network.Cookie, 0)
	for _, cookie := range cookies {
		if cookie.Domain == domain {
			newCookies = append(newCookies, cookie)
		}
	}

	cs := CookieConverter(newCookies)
	return cs, nil
}

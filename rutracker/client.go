package rutracker

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

var (
	ErrNotAuthorized = errors.New("not authorized: login first")
	ErrAuthFailed    = errors.New("incorrect username or password")
)

type Client struct {
	httpClient *http.Client
	username   string
	password   string
	authorized bool
	host       string
}

func NewClient(username, password string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		username: username,
		password: password,
		host:     Host,
	}
}

func (c *Client) Login() error {
	if c.username == "" || c.password == "" {
		return fmt.Errorf("rutracker credentials are required")
	}

	form := url.Values{}
	form.Set("login_username", c.username)
	form.Set("login_password", c.password)
	form.Set("login", "Вход")

	req, err := http.NewRequest(http.MethodPost, c.host+"/forum/login.php", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusFound {
		return ErrAuthFailed
	}

	hasSession := false
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "bb_session" && cookie.Value != "" {
			hasSession = true
			break
		}
	}
	if !hasSession {
		for _, setCookie := range resp.Header.Values("Set-Cookie") {
			if strings.Contains(setCookie, "bb_session=") {
				hasSession = true
				break
			}
		}
	}
	if !hasSession {
		return ErrAuthFailed
	}

	c.authorized = true
	return nil
}

func (c *Client) ensureAuth() error {
	if !c.authorized {
		return ErrNotAuthorized
	}
	return nil
}

func (c *Client) Search(query string) ([]Torrent, error) {
	if err := c.ensureAuth(); err != nil {
		return nil, err
	}

	body, err := c.postForm(c.host+"/forum/tracker.php", url.Values{"nm": {query}})
	if err != nil {
		return nil, err
	}
	if isLoginPage(body) {
		if err := c.Login(); err != nil {
			return nil, err
		}
		body, err = c.postForm(c.host+"/forum/tracker.php", url.Values{"nm": {query}})
		if err != nil {
			return nil, err
		}
	}

	return ParseSearch(body), nil
}

func (c *Client) GetThread(id string) (Thread, error) {
	if err := c.ensureAuth(); err != nil {
		return Thread{}, err
	}

	pageURL := fmt.Sprintf("%s/forum/viewtopic.php?t=%s", c.host, url.QueryEscape(id))
	body, err := c.get(pageURL)
	if err != nil {
		return Thread{}, err
	}
	if isLoginPage(body) {
		if err := c.Login(); err != nil {
			return Thread{}, err
		}
		body, err = c.get(pageURL)
		if err != nil {
			return Thread{}, err
		}
	}

	magnet := ParseMagnetLink(body)
	if magnet == "" {
		return Thread{}, fmt.Errorf("magnet link not found for topic %s", id)
	}

	title := parseThreadTitle(body)
	return Thread{
		ID:          id,
		Title:       title,
		Magnet:      magnet,
		Description: ParseDescription(body, 400),
	}, nil
}

func parseThreadTitle(body string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ""
	}
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if idx := strings.Index(title, " :: "); idx > 0 {
		title = title[:idx]
	}
	return title
}

func (c *Client) postForm(target string, form url.Values) (string, error) {
	req, err := http.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req)
}

func (c *Client) get(target string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) (string, error) {
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	decoded, err := decodeWindows1251(raw)
	if err != nil {
		return string(raw), nil
	}
	return decoded, nil
}

func decodeWindows1251(data []byte) (string, error) {
	reader := transform.NewReader(strings.NewReader(string(data)), charmap.Windows1251.NewDecoder())
	out, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func isLoginPage(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "login_password") && strings.Contains(lower, "login_username")
}

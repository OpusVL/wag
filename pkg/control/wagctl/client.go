package wagctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/NHAS/wag/internal/data"
	"github.com/NHAS/wag/internal/router"
	"github.com/NHAS/wag/pkg/control"
)

type CtrlClient struct {
	httpClient http.Client
}

// NewControlClient connects to the wag unix control socket specified by socketPath
func NewControlClient(socketPath string) *CtrlClient {
	return &CtrlClient{
		httpClient: http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}

func (c *CtrlClient) simplepost(path string, form url.Values) error {

	response, err := c.httpClient.Post("http://unix/"+path, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return errors.New(string(result))
	}

	return nil
}

// List devices, if the username field is empty (""), then list all devices. Otherwise list the one device corrosponding to the set username
func (c *CtrlClient) ListDevice(username string) (d []data.Device, err error) {

	response, err := c.httpClient.Get("http://unix/device/list?username=" + url.QueryEscape(username))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(string(result))
	}

	err = json.NewDecoder(response.Body).Decode(&d)

	return
}

// Take device address to remove
func (c *CtrlClient) DeleteDevice(address string) error {

	form := url.Values{}
	form.Add("address", address)

	return c.simplepost("device/delete", form)
}

func (c *CtrlClient) LockDevice(address string) error {

	form := url.Values{}
	form.Add("address", address)

	return c.simplepost("device/lock", form)
}

func (c *CtrlClient) UnlockDevice(address string) error {

	form := url.Values{}
	form.Add("address", address)

	return c.simplepost("device/unlock", form)
}

func (c *CtrlClient) ListAdminUsers(username string) (users []data.AdminModel, err error) {

	response, err := c.httpClient.Get("http://unix/webadmin/list?username=" + url.QueryEscape(username))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(string(result))
	}

	err = json.NewDecoder(response.Body).Decode(&users)

	return
}

// Take device address to remove
func (c *CtrlClient) AddAdminUser(username, password string) error {
	form := url.Values{}
	form.Add("username", username)
	form.Add("password", password)

	return c.simplepost("webadmin/add", form)
}

// Take device address to remove
func (c *CtrlClient) DeleteAdminUser(username string) error {
	form := url.Values{}
	form.Add("username", username)

	return c.simplepost("webadmin/delete", form)
}

func (c *CtrlClient) LockAdminUser(username string) error {
	form := url.Values{}
	form.Add("username", username)

	return c.simplepost("webadmin/lock", form)
}

func (c *CtrlClient) UnlockAdminUser(username string) error {

	form := url.Values{}
	form.Add("username", username)

	return c.simplepost("webadmin/unlock", form)
}

func (c *CtrlClient) ListUsers(username string) (users []data.UserModel, err error) {

	response, err := c.httpClient.Get("http://unix/users/list?username=" + url.QueryEscape(username))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(string(result))
	}

	err = json.NewDecoder(response.Body).Decode(&users)

	return
}

// Take device address to remove
func (c *CtrlClient) DeleteUser(username string) error {
	form := url.Values{}
	form.Add("username", username)

	return c.simplepost("users/delete", form)
}

func (c *CtrlClient) LockUser(username string) error {
	form := url.Values{}
	form.Add("username", username)

	return c.simplepost("users/lock", form)
}

func (c *CtrlClient) UnlockUser(username string) error {

	form := url.Values{}
	form.Add("username", username)

	return c.simplepost("users/unlock", form)
}

func (c *CtrlClient) ResetUserMFA(username string) error {

	form := url.Values{}
	form.Add("username", username)

	return c.simplepost("users/reset", form)
}

func (c *CtrlClient) Sessions() (out []string, err error) {

	response, err := c.httpClient.Get("http://unix/device/sessions")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(result, &out)

	return
}

func (c *CtrlClient) FirewallRules() (rules map[string]router.FirewallRules, err error) {

	response, err := c.httpClient.Get("http://unix/firewall/list")
	if err != nil {
		return rules, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return rules, err
		}

		return rules, errors.New("Error: " + string(result))
	}

	err = json.NewDecoder(response.Body).Decode(&rules)
	if err != nil {
		return rules, err
	}

	return
}

func (c *CtrlClient) ConfigReload() error {

	response, err := c.httpClient.Post("http://unix/config/reload", "text/plain", nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return errors.New(string(result))
	}

	return nil
}

func (c *CtrlClient) GetVersion() (string, error) {

	response, err := c.httpClient.Get("http://unix/version")
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func (c *CtrlClient) GetBPFVersion() (string, error) {

	response, err := c.httpClient.Get("http://unix/version/bpf")
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func (c *CtrlClient) Registrations() (result []control.RegistrationResult, err error) {

	response, err := c.httpClient.Get("http://unix/registration/list")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(string(result))
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, errors.New("unable to decode json: " + err.Error())
	}

	return
}

func (c *CtrlClient) NewRegistration(token, username, overwrite string, groups ...string) (r control.RegistrationResult, err error) {

	form := url.Values{}
	form.Add("username", username)
	form.Add("token", token)
	form.Add("overwrite", overwrite)

	for _, group := range groups {
		if !strings.HasPrefix(group, "group:") {
			return r, errors.New("group does not have 'group:' prefix: " + group)
		}
	}

	groupsJson, err := json.Marshal(groups)
	if err != nil {
		return r, err
	}

	form.Add("groups", string(groupsJson))

	response, err := c.httpClient.Post("http://unix/registration/create", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return control.RegistrationResult{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return control.RegistrationResult{}, err
		}

		return control.RegistrationResult{}, errors.New(string(result))
	}

	if err := json.NewDecoder(response.Body).Decode(&r); err != nil {
		return control.RegistrationResult{}, err
	}

	return
}

func (c *CtrlClient) DeleteRegistration(id string) (err error) {

	form := url.Values{}
	form.Add("id", id)

	return c.simplepost("registration/delete", form)
}

func (c *CtrlClient) Shutdown(cleanup bool) (err error) {

	form := url.Values{}
	form.Add("cleanup", fmt.Sprintf("%t", cleanup))

	return c.simplepost("shutdown", form)
}

func (c *CtrlClient) PinBPF() (err error) {

	response, err := c.httpClient.Get("http://unix/ebpf/pin")
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		return errors.New(string(result))
	}

	return
}

func (c *CtrlClient) UnpinBPF() (err error) {

	response, err := c.httpClient.Get("http://unix/ebpf/unpin")
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		result, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		return errors.New(string(result))
	}

	return
}
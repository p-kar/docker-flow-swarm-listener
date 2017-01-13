package main

// TODO: Document service labels com.df.notify, com.df.notifyBody, and com.df.notifyMethod
// TODO: Document env. vars. DF_SLACK_URL, DF_SLACK_CHANNEL, DF_SLACK_USERNAME, DF_SLACK_TEXT, DF_SLACK_ICON_EMOJI
// TODO: Use labels instead env. vars. when present
// TODO: Document service labels com.df.slackUrl, com.df.slackChannel, com.df.slackUsername, com.df.slackText, com.df.slackIconEmoji
// TODO: Write an article
import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"bytes"
	"encoding/json"
	"html/template"
)

var logPrintf = log.Printf

type Service struct {
	DockerClient          *client.Client
	Host                  string
	NotifCreateServiceUrl string
	NotifRemoveServiceUrl string
	Services              map[string]bool
	ServiceLastCreatedAt  time.Time
	Slack                 Slack
}

// TODO: Add to NewServiceFromEnv
// TODO: Overwrite with service labels
type Slack struct {
	Url       string
	Channel   string
	Username  string
	Text      string
	IconEmoji string
}

type Servicer interface {
	GetServices() ([]swarm.Service, error)
	GetNewServices(services []swarm.Service) ([]swarm.Service, error)
	NotifyServicesCreate(services []swarm.Service, retries, interval int) error
	NotifyServicesRemove(services []string, retries, interval int) error
}

func (m *Service) GetServices() ([]swarm.Service, error) {
	filter := filters.NewArgs()
	filter.Add("label", "com.df.notify=true")
	services, err := m.DockerClient.ServiceList(context.Background(), types.ServiceListOptions{Filters: filter})
	if err != nil {
		logPrintf(err.Error())
		return []swarm.Service{}, err
	}
	return services, nil
}

func (m *Service) GetNewServices(services []swarm.Service) ([]swarm.Service, error) {
	newServices := []swarm.Service{}
	tmpCreatedAt := m.ServiceLastCreatedAt
	for _, s := range services {
		if tmpCreatedAt.Nanosecond() == 0 || s.Meta.CreatedAt.After(tmpCreatedAt) {
			newServices = append(newServices, s)
			m.Services[s.Spec.Name] = true
			if m.ServiceLastCreatedAt.Before(s.Meta.CreatedAt) {
				m.ServiceLastCreatedAt = s.Meta.CreatedAt
			}
		}
	}
	return newServices, nil
}

func (m *Service) GetRemovedServices(services []swarm.Service) []string {
	tmpMap := make(map[string]bool)
	for k, _ := range m.Services {
		tmpMap[k] = true
	}
	for _, v := range services {
		if _, ok := m.Services[v.Spec.Name]; ok {
			delete(tmpMap, v.Spec.Name)
		}
	}
	rs := []string{}
	for k, _ := range tmpMap {
		rs = append(rs, k)
	}
	return rs
}

func (m *Service) NotifyServicesCreate(services []swarm.Service, retries, interval int) error {
	errs := []error{}
	for _, s := range services {
		if err := m.notifyServiceCreateGeneric(s, retries, interval); err != nil {
			errs = append(errs, err)
		}
		if err := m.notifyServiceCreateSlack(s, retries, interval); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("At least one request produced errors. Please consult logs for more details.")
	}
	return nil
}

func (m *Service) notifyServiceCreateSlack(service swarm.Service, retries, interval int) error {
	if len(m.Slack.Url) > 0 {
		slack := m.getSlackData(service)
		_, err := url.Parse(slack.Url)
		if err != nil {
			logPrintf("ERROR: %s", err.Error())
			return err
		}
		js, _ := json.Marshal(slack)
		err = m.sendRequest(slack.Url, "POST", js, retries, interval)
		return err
	}
	return nil
}

func (m *Service) getSlackData(service swarm.Service) Slack {
	slack := m.Slack
	type Data struct {
		ServiceName string
	}
	data := Data{
		ServiceName: service.Spec.Name,
	}
	var content bytes.Buffer
	tmpl, _ := template.New("contentTemplate").Parse(slack.Text)
	tmpl.Execute(&content, data)
	slack.Text = content.String()
	return slack
}

func (m *Service) notifyServiceCreateGeneric(service swarm.Service, retries, interval int) error {
	if len(m.NotifCreateServiceUrl) > 0 {
		if _, ok := service.Spec.Labels["com.df.notify"]; ok {
			body := ""
			method := "GET"
			urlObj, err := url.Parse(m.NotifCreateServiceUrl)
			if err != nil {
				logPrintf("ERROR: %s", err.Error())
				return err
			}
			parameters := url.Values{}
			parameters.Add("serviceName", service.Spec.Name)
			for k, v := range service.Spec.Labels {
				if strings.HasPrefix(k, "com.df") {
					if k == "com.df.notifyBody" {
						body = v
					} else if k == "com.df.notifyMethod" {
						method = v
					} else if k != "com.df.notify" {
						parameters.Add(strings.TrimPrefix(k, "com.df."), v)
					}
				}
			}
			urlObj.RawQuery = parameters.Encode()
			return m.sendRequest(urlObj.String(), method, []byte(body), retries, interval)
		}
	}
	return nil
}

func (m *Service) sendRequest(url, method string, msg []byte, retries, interval int) error {
	logPrintf("Sending service created notification to %s", url)
	client := http.Client{}
	for i := 1; i <= retries; i++ {
		req, _ := http.NewRequest(method, url, bytes.NewBuffer(msg))
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		} else if i < retries {
			if interval > 0 {
				t := time.NewTicker(time.Second * time.Duration(interval))
				<-t.C
			}
		} else {
			if err != nil {
				logPrintf("ERROR: %s", err.Error())
				return err
			} else if resp.StatusCode != http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				msg := fmt.Errorf("Request %s returned status code %d\n%s", url, resp.StatusCode, string(body[:]))
				logPrintf("ERROR: %s", msg)
				return msg
			}
		}
	}
	return nil
}

func (m *Service) NotifyServicesRemove(services []string, retries, interval int) error {
	errs := []error{}
	for _, v := range services {
		urlObj, err := url.Parse(m.NotifRemoveServiceUrl)
		if err != nil {
			logPrintf("ERROR: %s", err.Error())
			errs = append(errs, err)
			break
		}
		parameters := url.Values{}
		parameters.Add("serviceName", v)
		parameters.Add("distribute", "true")
		urlObj.RawQuery = parameters.Encode()
		fullUrl := urlObj.String()
		logPrintf("Sending service removed notification to %s", fullUrl)
		for i := 1; i <= retries; i++ {
			resp, err := http.Get(fullUrl)
			if err == nil && resp.StatusCode == http.StatusOK {
				delete(m.Services, v)
				break
			} else if i < retries {
				if interval > 0 {
					t := time.NewTicker(time.Second * time.Duration(interval))
					<-t.C
				}
			} else {
				if err != nil {
					logPrintf("ERROR: %s", err.Error())
					errs = append(errs, err)
				} else if resp.StatusCode != http.StatusOK {
					msg := fmt.Errorf("Request %s returned status code %d", fullUrl, resp.StatusCode)
					logPrintf("ERROR: %s", msg)
					errs = append(errs, msg)
				}
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("At least one request produced errors. Please consult logs for more details.")
	}
	return nil
}

func NewService(host, notifCreateServiceUrl, notifRemoveServiceUrl string, slack Slack) *Service {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	dc, err := client.NewClient(host, "v1.22", nil, defaultHeaders)
	if err != nil {
		logPrintf(err.Error())
	}
	if len(slack.Url) > 0 && !strings.HasPrefix(slack.Url, "http") {
		slack.Url = fmt.Sprintf("https://%s", slack.Url)
	}
	return &Service{
		Host: host,
		NotifCreateServiceUrl: notifCreateServiceUrl,
		NotifRemoveServiceUrl: notifRemoveServiceUrl,
		Services:              make(map[string]bool),
		DockerClient:          dc,
		Slack:                 slack,
	}
}

func NewServiceFromEnv() *Service {
	host := "unix:///var/run/docker.sock"
	if len(os.Getenv("DF_DOCKER_HOST")) > 0 {
		host = os.Getenv("DF_DOCKER_HOST")
	}
	notifCreateServiceUrl := os.Getenv("DF_NOTIF_CREATE_SERVICE_URL")
	if len(notifCreateServiceUrl) == 0 {
		notifCreateServiceUrl = os.Getenv("DF_NOTIFICATION_URL")
	}
	notifRemoveServiceUrl := os.Getenv("DF_NOTIF_REMOVE_SERVICE_URL")
	if len(notifRemoveServiceUrl) == 0 {
		notifRemoveServiceUrl = os.Getenv("DF_NOTIFICATION_URL")
	}
	slack := Slack{
		Url: os.Getenv("DF_SLACK_URL"),
		Channel: os.Getenv("DF_SLACK_CHANNEL"),
		Username: os.Getenv("DF_SLACK_USERNAME"),
		Text: os.Getenv("DF_SLACK_TEXT"),
		IconEmoji: os.Getenv("DF_SLACK_ICON_EMOJI"),
	}
	return NewService(host, notifCreateServiceUrl, notifRemoveServiceUrl, slack)
}

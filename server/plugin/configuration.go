package plugin

import (
	"html/template"
	"strings"

	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
type configuration struct {
	ExcludedChannelString string
	excludedChannels      []string
	Text                  string
	text                  *template.Template
}

// OnConfigurationChange loads the plugin configuration, validates it and saves it.
func (p *Plugin) OnConfigurationChange() error {
	configuration := new(configuration)

	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "Failed to load plugin configuration")
	}

	if configuration.Text == "" {
		return errors.New("Empty Text not allowed")
	}

	if strings.Index(configuration.Text, "{{.Channels}}") == -1 {
		return errors.New("Text must have `{{.Channels}}`")
	}

	t, err := template.New("t").Parse(configuration.Text)
	if err != nil {
		return errors.New(err.Error())
	}
	configuration.text = t
	configuration.excludedChannels = strings.Split(configuration.ExcludedChannelString, " ")

	p.setConfiguration(configuration)
	return nil
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}
	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		panic("setConfiguration called with the existing configuration")
	}
	p.configuration = configuration
}

package service

import (
	"html/template"
	"sync"

	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// CollectionTemplate is the database collection where Templates are stored
const CollectionTemplate = "Template"

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	Factory   *Factory
	Sources   []TemplateSource
	Templates map[string]*model.Template
	Updates   chan model.Template
	mutex     sync.RWMutex
}

// Start is meant to be run as a goroutine, and constantly monitors the "Updates" channel for
// news that a template has been updated.
func (service *Template) Start() {

	for {
		template := <-service.Updates

		service.Cache(&template)

		realtimeBroker := service.Factory.RealtimeBroker()
		streamService := service.Factory.Stream()

		iterator, err := streamService.ListByTemplate(template.TemplateID)

		if err != nil {
			derp.Report(derp.Wrap(err, "ghost.service.Realtime", "Error Listing Streams for Template", template))
			return
		}

		var stream model.Stream

		for iterator.Next(&stream) {
			realtimeBroker.streamUpdates <- stream
		}
	}
}

func (service *Template) AddSource(source TemplateSource) error {

	service.Sources = append(service.Sources, source)

	list, err := source.List()

	if err != nil {
		return derp.Wrap(err, "ghost.service.Template.AddSource", "Error listing templates from", source)
	}

	// Iterate through every template
	for _, name := range list {

		template, err := source.Load(name)

		if err != nil {
			return derp.Wrap(err, "ghost.service.Template.AddSource", "Error loading template", name)
		}

		// Save the template in memory.
		service.Cache(template)
	}

	// Watch for changes to this TemplateSource
	source.Watch(service.Updates)

	return nil
}

// List returns an iterator containing all of the Templates who match the provided criteria
func (service *Template) List(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return nil, derp.New(500, "ghost.service.Template.List", "Unimplemented")
}

// Load retrieves an Template from the database
func (service *Template) Load(templateID string) (*model.Template, error) {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	// Look in the local cache first
	if template, ok := service.Templates[templateID]; ok {
		return template, nil
	}

	// Otherwise, search all sources for the Template.
	for index := range service.Sources {
		if template, err := service.Sources[index].Load(templateID); err != nil {
			service.Templates[templateID] = template
			return template, nil
		}
	}

	return nil, derp.New(404, "ghost.sevice.Template.Load", "Could not load Template", templateID, service.Names())
}

// LoadCompiled returns the compiled template for the requested arguments.
func (service *Template) LoadCompiled(templateID string, stateName string, viewName string) (*model.Template, *template.Template, error) {

	template, err := service.Load(templateID)

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.service.Template.LoadCompiled", "Error loading template")
	}

	state, err := template.View(stateName, viewName)

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.service.Template.LoadCompiled", "Error loading state")
	}

	view, err := state.Compiled()

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.service.Template.LoadCompiled", "Error getting compiled template")
	}

	clone, err := view.Clone()

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.service.Template.LoadCompiled", "Error cloning template")
	}

	return template, clone, nil
}

// Cache adds/updates an Template in the memory cache
func (service *Template) Cache(template *model.Template) {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.Templates[template.TemplateID] = template
}

// Delete removes an Template from the database (virtual delete)
func (service *Template) Delete(template model.Template, note string) error {
	delete(service.Templates, template.TemplateID)
	return nil
}

func (service *Template) Names() []string {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	result := make([]string, len(service.Templates))

	index := 0
	for key := range service.Templates {
		result[index] = key
		index = index + 1
	}

	return result
}

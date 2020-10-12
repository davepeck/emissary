package render

import (
	"bytes"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// Folder renderer can
type Folder struct {
	layoutService   LayoutService
	folderService   FolderService
	templateService TemplateService
	streamService   StreamService
	folder          model.Folder
	view            string
}

// NewFolder returns a fully initialized Folder renderer
func NewFolder(layoutService LayoutService, folderService FolderService, templateService TemplateService, streamService StreamService, folder model.Folder, view string) Folder {

	return Folder{
		layoutService:   layoutService,
		folderService:   folderService,
		templateService: templateService,
		streamService:   streamService,
		folder:          folder,
		view:            view,
	}
}

// Token returns the token of this folder.
func (w Folder) Token() string {
	return w.folder.Token
}

// Label returns the human-friendly label for this folder.
func (w Folder) Label() string {
	return w.folder.Label
}

func (w Folder) Description() string {
	return w.folder.Description
}

// Render returns an HTML representation of this Folder.
func (w Folder) Render() (string, error) {

	layout := w.layoutService.Layout()

	var buffer bytes.Buffer

	if err := layout.ExecuteTemplate(&buffer, "folder", w); err != nil {
		return "", derp.Wrap(err, "ghost.render.Folder.Render", "Error executing Template", layout)
	}

	return buffer.String(), nil
}

func (w Folder) Folders() ([]FolderListItem, error) {

	folders, err := w.folderService.ListNested()

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.Stream.AllFolders", "Error retrieving all folders")
	}

	return NewFolderList(folders), nil
}

// SubFolders returns renderers for all of the SubFolders within the current Folder.
func (w Folder) SubFolders() ([]FolderListItem, error) {

	folders, err := w.folderService.ListByParent(w.folder.FolderID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.Folder.SubFolders", "Error retrieving sub-folders", w.folder)
	}

	result := []FolderListItem{}
	folder := w.folderService.New()

	for folders.Next(folder) {
		result = append(result, NewFolderListItem(*folder))
	}

	return result, nil
}

// Streams returns renderers for all Streams contained within this folder.
func (w Folder) Streams(view string) ([]Stream, error) {

	var result []Stream

	it, err := w.streamService.ListByFolder(w.folder.FolderID)

	if err != nil {
		return result, derp.Wrap(err, "ghost.render.Folder.Streams", "Error listing streams in folder", w.folder)
	}

	stream := w.streamService.New()

	for it.Next(stream) {
		result = append(result, NewStream(w.layoutService, w.folderService, w.templateService, w.streamService, *stream, view))
	}

	return result, nil
}

// SubTemplates returns an array of templates that can be placed inside this Folder
func (w Folder) SubTemplates() ([]model.Template) {
	return w.templateService.ListByContainer("folder")
}

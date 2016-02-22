package templating // import "github.com/cafebazaar/blacksmith/templating"

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/logging"
)

const (
	templatesDebugTag = "TEMPLATING"
)

func findFiles(path string) ([]string, error) {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for i := range infos {
		if !infos[i].IsDir() && infos[i].Name()[0] != '.' {
			files = append(files, infos[i].Name())
		}
	}
	return files, nil
}

//FromPath creates templates from the files located in the specifed path
func templateFromPath(tmplPath string) (*template.Template, error) {
	files, err := findFiles(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("Error while trying to list files in%s: %s", tmplPath, err)
	}

	t := template.New("")
	t.Delims("<<", ">>")
	t.Funcs(map[string]interface{}{
		"V": func(key string) string {
			return ""
		},
		"S": func(key string, value string) string {
			return ""
		},
		"VD": func(key string) string {
			return ""
		},
		"D": func(key string) string {
			return ""
		},
		"b64": func(text string) string {
			return ""
		},
		"b64template": func(templateName string) string {
			return ""
		},
	})

	for i := range files {
		files[i] = path.Join(tmplPath, files[i])
	}

	t, err = t.ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func executeTemplate(rootTemplte *template.Template, templateName string, machine datasource.Machine) (string, error) {
	template := rootTemplte.Lookup(templateName)

	if template == nil {
		return "", fmt.Errorf("template with name=%s wasn't found for root=%s",
			templateName, rootTemplte)
	}

	buf := new(bytes.Buffer)
	template.Funcs(map[string]interface{}{
		"V": func(key string) string {
			flag, err := machine.GetFlag(key)
			if err != nil { // TODO excepts Not-Found
				logging.Log(templatesDebugTag,
					"Error while getting flag key=%s for machine=%s: %s",
					key, machine.Name(), err)
				return ""
			}
			return flag
		},
		"S": func(key string, value string) string {
			err := machine.SetFlag(key, value)
			if err != nil {
				logging.Log(templatesDebugTag,
					"Error while setting flag key=%s value=%s for machine=%s: %s",
					key, value, machine.Name(), err)
			}
			return ""
		},
		"VD": func(key string) string {
			flag, err := machine.GetAndDeleteFlag(key)
			if err != nil { // TODO excepts Not-Found
				logging.Log(templatesDebugTag,
					"Error while get+deleting flag key=%s for machine=%s: %s",
					key, machine.Name(), err)
				return ""
			}
			return flag
		},
		"D": func(key string) string {
			err := machine.DeleteFlag(key)
			if err != nil {
				logging.Log(templatesDebugTag,
					"Error while deleting flag key=%s for machine=%s: %s",
					key, machine.Name(), err)
			}
			return ""
		},
		"b64": func(text string) string {
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"b64template": func(templateName string) string {
			text, err := executeTemplate(rootTemplte, templateName, machine)
			if err != nil {
				logging.Log(templatesDebugTag,
					"Error while b64template for templateName=%s machine=%s: %s",
					templateName, machine.Name(), err)
				return ""
			}
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
	})
	err := template.ExecuteTemplate(buf, templateName, nil)
	if err != nil {
		return "", err
	}
	str := buf.String()
	str = strings.Trim(str, "\n")
	return str, nil
}

func ExecuteTemplateFolder(tmplFolder string, machine datasource.Machine) (string, error) {
	template, err := templateFromPath(tmplFolder)
	if err != nil {
		return "", fmt.Errorf("Error while reading the template with path=%s: %s",
			tmplFolder, err)
	}

	return executeTemplate(template, "main", machine)
}

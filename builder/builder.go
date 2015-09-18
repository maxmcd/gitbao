package builder

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/maxmcd/gitbao/config"
	"github.com/maxmcd/gitbao/logger"
	"github.com/maxmcd/gitbao/model"
)

var github_access_key string

func init() {
	github_access_key = os.Getenv("GITHUB_GIST_ACCESS_KEY")
	if github_access_key == "" {
		panic("Github access key required")
	}
}

func Build(gistId, cfg string, l logger.Log) (err error, name string) {

	configStruct, err := config.Parse(cfg)
	if err != nil {
		l.Write(err.Error())
	}

	l.Write("Fetching gist data")
	gist, err := FetchGistData(gistId)
	if err != nil {
		return
	}

	fileCount := len(gist.Files)
	l.Write("%d files found:", fileCount)

	var hasGoFiles bool

	for _, file := range gist.Files {
		filenameParts := strings.Split(file.Filename, ".")
		if filenameParts[len(filenameParts)-1] == "go" {
			hasGoFiles = true
		}
		l.Write("&nbsp;&nbsp;- %s", file.Filename)
	}

	if hasGoFiles != true {
		l.Write("No Go files found in this gist. Exiting.")
		return
	}

	l.Write("Downloading gist contents for build")
	directory, err := DownloadFromRepo(gist.GitPullURL)
	l.Write("Files downloaded in directory: %s", directory)

	err = GoBuild(gist, directory, l)
	if err != nil {
		return
	}
	l.Write("Build successful")

	l.Write("Zipping contents")
	err = CreateZip(gist, configStruct, directory)
	if err != nil {
		return
	}

	l.Write("Uploading packaged contents")
	err = CreateLambda(directory)
	if err != nil {
		return
	}

	id, err := model.CreateBao(gistId, directory)
	if err != nil {
		return
	}
	l.Write("Bao successfully published at:")
	l.Write("%s.gitbao.com", id.Hex())

	l.Write("cleaning up")
	err = os.RemoveAll(directory)
	if err != nil {
		return
	}
	err = os.Remove("bin" + directory)
	if err != nil {
		return
	}
	err = os.Remove(directory + ".zip")
	if err != nil {
		return
	}

	fmt.Println(err)

	return
}

func CreateLambda(directory string) error {

	zipBytes, err := ioutil.ReadFile(directory + ".zip")
	if err != nil {
		return err
	}

	svc := lambda.New(&aws.Config{Region: aws.String("us-east-1")})

	params := &lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: zipBytes,
		},
		FunctionName: aws.String(directory),                                               // Required
		Handler:      aws.String("handler_example.handler"),                               // Required
		Role:         aws.String("arn:aws:iam::651778473396:role/lambda_basic_execution"), // Required
		Runtime:      aws.String("nodejs"),                                                // Required
		// Description:  aws.String("nodejs"),
		MemorySize: aws.Int64(150),
		Timeout:    aws.Int64(3),
	}
	resp, err := svc.CreateFunction(params)
	_ = resp
	if err != nil {
		return err
	}
	return nil
}

func CreateZip(gist GithubGist, cfg config.Config, directory string) error {

	buf := new(bytes.Buffer)
	// Create a new zip archive.
	w := zip.NewWriter(buf)

	err := addFileToZip(w, "bin"+directory, "userapp")
	if err != nil {
		return err
	}

	err = addHandlerToZip(w, cfg)
	if err != nil {
		return err
	}

	for _, file := range gist.Files {
		err = addFileToZip(w, directory+"/"+file.Filename, "")
	}

	err = w.Close()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(directory+".zip", buf.Bytes(), 777)
	if err != nil {
		return err
	}

	return nil
}

func addHandlerToZip(w *zip.Writer, cfg config.Config) error {
	path := "lambda/handler_example.js"
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	wr, err := w.CreateHeader(header)
	if err != nil {
		return err
	}
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	handlerTemplate := string(fileBytes)
	t := template.Must(template.New("handler").Parse(handlerTemplate))
	err = t.Execute(wr, cfg)
	if err != nil {
		return err
	}
	return nil
}

func addFileToZip(w *zip.Writer, path string, newname string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	if newname != "" {
		header.Name = newname
	}
	wr, err := w.CreateHeader(header)
	if err != nil {
		return err
	}
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = wr.Write(fileBytes)
	if err != nil {
		return err
	}
	return nil
}

func GoBuild(gist GithubGist, directory string, l logger.Log) error {

	arguments := []string{"build", "-o", "bin" + directory}

	for _, file := range gist.Files {
		filenameParts := strings.Split(file.Filename, ".")
		if filenameParts[len(filenameParts)-1] == "go" {
			arguments = append(arguments, directory+"/"+file.Filename)
		}
	}

	os.Setenv("GOOS", "linux")
	os.Setenv("GOARCH", "amd64")
	cmd := exec.Command("go", arguments...)
	byteOut, err := cmd.CombinedOutput()

	if len(byteOut) > 0 {
		l.Write("There was an error building this Go application:")
		output := string(byteOut)
		fmt.Println(output)
		output = strings.Replace(output, directory, "", -1)
		output = strings.Replace(output, "\n", "<br>", -1)
		l.Write(output)
		return fmt.Errorf("error building")
	}
	if err != nil {
		return err
	}
	return nil
}

func FetchGistData(gistId string) (gist GithubGist, err error) {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET",
		"https://api.github.com/gists/"+gistId,
		nil,
	)
	if err != nil {
		return
	}
	req.SetBasicAuth(
		github_access_key,
		"",
	)
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error fetching gist data")
		return
	}
	if resp.StatusCode == 404 {
		err = fmt.Errorf("404 gist not found")
		return
	}
	if err != nil {
		return
	}

	err = json.Unmarshal(contents, &gist)
	if err != nil {
		return
	}
	return
}

func DownloadFromRepo(gitPullUrl string) (directory string, err error) {
	path := "."
	directory, err = ioutil.TempDir(path, "bao")
	if err != nil {
		return
	}
	cmd := exec.Command("git", "clone", gitPullUrl, path+"/"+directory)
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	cmd.Wait()

	return
}

type GithubGistFile struct {
	Content   string `json:"content"`
	Filename  string `json:"filename"`
	Language  string `json:"language"`
	RawURL    string `json:"raw_url"`
	Size      int    `json:"size"`
	Truncated bool   `json:"truncated"`
	Type      string `json:"type"`
}

type GithubGist struct {
	Comments    int                       `json:"comments"`
	CommentsURL string                    `json:"comments_url"`
	CommitsURL  string                    `json:"commits_url"`
	CreatedAt   string                    `json:"created_at"`
	Description string                    `json:"description"`
	Files       map[string]GithubGistFile `json:"files"`
	Forks       []interface{}             `json:"forks"`
	ForksURL    string                    `json:"forks_url"`
	GitPullURL  string                    `json:"git_pull_url"`
	GitPushURL  string                    `json:"git_push_url"`
	History     []struct {
		ChangeStatus struct {
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
			Total     int `json:"total"`
		} `json:"change_status"`
		CommittedAt string `json:"committed_at"`
		URL         string `json:"url"`
		User        struct {
			AvatarURL         string `json:"avatar_url"`
			EventsURL         string `json:"events_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			GravatarID        string `json:"gravatar_id"`
			HTMLURL           string `json:"html_url"`
			ID                int    `json:"id"`
			Login             string `json:"login"`
			OrganizationsURL  string `json:"organizations_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			ReposURL          string `json:"repos_url"`
			SiteAdmin         bool   `json:"site_admin"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			Type              string `json:"type"`
			URL               string `json:"url"`
		} `json:"user"`
		Version string `json:"version"`
	} `json:"history"`
	HTMLURL string `json:"html_url"`
	ID      string `json:"id"`
	Owner   struct {
		AvatarURL         string `json:"avatar_url"`
		EventsURL         string `json:"events_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		GravatarID        string `json:"gravatar_id"`
		HTMLURL           string `json:"html_url"`
		ID                int    `json:"id"`
		Login             string `json:"login"`
		OrganizationsURL  string `json:"organizations_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		ReposURL          string `json:"repos_url"`
		SiteAdmin         bool   `json:"site_admin"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		Type              string `json:"type"`
		URL               string `json:"url"`
	} `json:"owner"`
	Public    bool        `json:"public"`
	UpdatedAt string      `json:"updated_at"`
	URL       string      `json:"url"`
	User      interface{} `json:"user"`
}

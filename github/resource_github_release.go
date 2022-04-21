package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/v43/github"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
	"strconv"
)

func resourceGithubRelease() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubReleaseCreateUpdate,
		Update: resourceGithubReleaseCreateUpdate,
		Read:   resourceGithubReleaseRead,
		Delete: resourceGithubReleaseDelete,
		Importer: &schema.ResourceImporter{
			State: resourceGithubReleaseImport,
		},

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tag_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"target_commitish": {
				Type:     schema.TypeString,
				Default:  "main",
				Optional: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"body": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"draft": {
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
				ForceNew: true,
			},
			"prerelease": {
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},
			"generate_release_notes": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
			"discussion_category_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGithubReleaseCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	ctx := context.Background()
	if !d.IsNewResource() {
		ctx = context.WithValue(ctx, ctxId, d.Id())
	}

	client := meta.(*Owner).v3client
	owner := meta.(*Owner).name
	repoName := d.Get("repository").(string)
	tagName := d.Get("tag_name").(string)
	targetCommitish := d.Get("target_commitish").(string)
	draft := d.Get("draft").(bool)
	prerelease := d.Get("prerelease").(bool)
	generateReleaseNotes := d.Get("generate_release_notes").(bool)

	req := &github.RepositoryRelease{
		TagName:              github.String(tagName),
		TargetCommitish:      github.String(targetCommitish),
		Draft:                github.Bool(draft),
		Prerelease:           github.Bool(prerelease),
		GenerateReleaseNotes: github.Bool(generateReleaseNotes),
	}

	if v, ok := d.GetOk("body"); ok {
		req.Body = github.String(v.(string))
	}

	if v, ok := d.GetOk("name"); ok {
		req.Name = github.String(v.(string))
	}

	if v, ok := d.GetOk("discussion_category_name"); ok {
		req.DiscussionCategoryName = github.String(v.(string))
	}

	var release *github.RepositoryRelease
	var resp *github.Response
	var err error
	if d.IsNewResource() {
		log.Printf("[DEBUG] Creating release: %s (%s/%s)",
			targetCommitish, owner, repoName)
		release, resp, err = client.Repositories.CreateRelease(ctx, owner, repoName, req)
		if resp != nil {
			log.Printf("[DEBUG] Response from creating release: %#v", *resp)
		}
	} else {
		number := d.Get("number").(int64)
		log.Printf("[DEBUG] Updating release: %d:%s (%s/%s)",
			number, targetCommitish, owner, repoName)
		release, resp, err = client.Repositories.EditRelease(ctx, owner, repoName, number, req)
		if resp != nil {
			log.Printf("[DEBUG] Response from updating release: %#v", *resp)
		}
	}

	if err != nil {
		return err
	}
	transformResponseToResourceData(d, release, repoName)
	return nil
}

func resourceGithubReleaseRead(d *schema.ResourceData, meta interface{}) error {
	repository := d.Get("repository").(string)
	ctx := context.WithValue(context.Background(), ctxId, d.Id())
	client := meta.(*Owner).v3client
	owner := meta.(*Owner).name
	releaseID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}
	if releaseID == 0 {
		return fmt.Errorf("`release_id` must be present")
	}

	release, _, err := client.Repositories.GetRelease(ctx, owner, repository, releaseID)
	if err != nil {
		return err
	}
	transformResponseToResourceData(d, release, repository)
	return nil
}

func resourceGithubReleaseDelete(d *schema.ResourceData, meta interface{}) error {
	ctx := context.WithValue(context.Background(), ctxId, d.Id())
	repository := d.Get("repository").(string)
	client := meta.(*Owner).v3client
	owner := meta.(*Owner).name

	releaseIDStr := d.Id()
	releaseID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(releaseIDStr, err)
	}
	if releaseID == 0 {
		return fmt.Errorf("`release_id` must be present")
	}

	_, err = client.Repositories.DeleteRelease(ctx, owner, repository, releaseID)
	if err != nil {
		return fmt.Errorf("error deleting GitHub release reference %s/%s (%s): %s",
			fmt.Sprint(releaseID), repository, owner, err)
	}
	return nil
}

func resourceGithubReleaseImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	repoName, releaseIDStr, err := parseTwoPartID(d.Id(), "repository", "release")
	if err != nil {
		return []*schema.ResourceData{d}, err
	}

	releaseID, err := strconv.ParseInt(releaseIDStr, 10, 64)
	if err != nil {
		return []*schema.ResourceData{d}, unconvertibleIdErr(releaseIDStr, err)
	}
	if releaseID == 0 {
		return []*schema.ResourceData{d}, fmt.Errorf("`release_id` must be present")
	}
	log.Printf("[DEBUG] Importing release with ID: %d, for repository: %s", releaseID, repoName)

	client := meta.(*Owner).v3client
	owner := meta.(*Owner).name
	ctx := context.Background()
	repository, _, err := client.Repositories.Get(ctx, owner, repoName)
	if repository == nil || err != nil {
		return []*schema.ResourceData{d}, err
	}
	d.Set("repository", *repository.Name)

	release, _, err := client.Repositories.GetRelease(ctx, owner, *repository.Name, releaseID)
	if release == nil || err != nil {
		return []*schema.ResourceData{d}, err
	}
	d.SetId(strconv.FormatInt(release.GetID(), 10))

	return []*schema.ResourceData{d}, nil
}

func transformResponseToResourceData(d *schema.ResourceData, release *github.RepositoryRelease, repository string) {
	d.SetId(strconv.FormatInt(release.GetID(), 10))
	d.Set("release_id", release.GetID())
	d.Set("node_id", release.GetNodeID())
	d.Set("repository", repository)
	d.Set("tag_name", release.GetTagName())
	d.Set("target_commitish", release.GetTargetCommitish())
	d.Set("name", release.GetName())
	d.Set("body", release.GetBody())
	d.Set("draft", release.GetDraft())
	d.Set("generate_release_notes", release.GetGenerateReleaseNotes())
	d.Set("prerelease", release.GetPrerelease())
	d.Set("discussion_category_name", release.GetDiscussionCategoryName())
	d.Set("created_at", release.GetCreatedAt())
	d.Set("published_at", release.GetPublishedAt())
	d.Set("url", release.GetURL())
	d.Set("html_url", release.GetHTMLURL())
	d.Set("assets_url", release.GetAssetsURL())
	d.Set("upload_url", release.GetUploadURL())
	d.Set("zipball_url", release.GetZipballURL())
	d.Set("tarball_url", release.GetTarballURL())
}

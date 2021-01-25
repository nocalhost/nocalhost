package req

func (p *SyncthingHttpClient) FolderOverride() error {
	_, err := p.Post("rest/db/override?folder="+p.folderName, "")
	if err != nil {
		return err
	}
	return nil
}

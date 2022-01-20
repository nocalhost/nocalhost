package req

func (p *SyncthingHttpClient) Scan() error {
	_, err := p.Post("rest/db/scan?folder="+p.folderName, "")
	if err != nil {
		return err
	}
	return nil
}

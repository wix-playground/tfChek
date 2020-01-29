package launcher

func normalizeGitRemotes(remotes *[]string) *[]string {
	if remotes == nil || len(*remotes) == 0 {
		return nil
	}
	r := *remotes
	var nr *[]string
	//Need to maintain override order here and remove duplicates
	for i := range *remotes {
		nr = prepend2normal(r[len(r)-1-i], nr)
	}
	return nr
}

func prepend2normal(r string, normal *[]string) *[]string {
	var contains bool = false
	if normal == nil {
		return &[]string{r}
	}
	for _, e := range *normal {
		if e == r {
			contains = true
			break
		}
	}
	if !contains {
		n := append([]string{r}, *normal...)
		return &n
	}
	return normal
}

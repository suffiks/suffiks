package controllers

func mergeMaps(maps ...map[string]string) map[string]string {
	ret := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			ret[k] = v
		}
	}
	return ret
}

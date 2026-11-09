package httpx

import "github.com/scc-platform/shared/lists"

// FirstError возвращает первую непустую ошибку из списка.
func FirstError(errs ...error) error {
	found, _ := lists.Find(errs, notNil)
	return found
}

func notNil(err error) bool {
	return err != nil
}

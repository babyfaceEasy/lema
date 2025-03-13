package routes

import (
	"github.com/gorilla/mux"
)

func registerRepositoryRoutes(grp *mux.Router) {
	router := grp.PathPrefix("/repositories").Subrouter()

	// Example endpoint with annotations
    // getCommits godoc
    // @Summary Retrieve commits
    // @Description Get commits of a repository
    // @Produce json
    // @Param repositoryName path string true "Repository Name"
    // @Success 200 {array} Commit
    // @Router /repositories/{repository_name}/commits [get]
	router.HandleFunc("/{repository_name}/commits", handler.GetRepositoryCommits).Methods("GET")
}

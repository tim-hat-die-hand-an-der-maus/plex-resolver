package main

func directoriesToMovies(videos []Directory) []Movie {
	var movies []Movie

	for _, video := range videos {
		movies = append(movies, video.ToMovie())
	}

	return movies
}

func videosToMovies(videos []Video) []Movie {
	var movies []Movie

	for _, video := range videos {
		movies = append(movies, video.ToMovie())
	}

	return movies
}

func contains[T comparable](elements []T, value T) bool {
	for _, element := range elements {
		if element == value {
			return true
		}
	}

	return false
}

func isValidDirectoryName(name string) bool {
	validNames := []string{"show", "movie"}

	return contains(validNames, name)
}

Example use: `go run cmd/igscraper/main.go zuck`

The program will recursively look through all the posts, filter out videos (reels included), and save all the images to a folder named after the IG profile.

Currently the program will stop executing after ~160 images due to rate limits that are not handled in the code. The code needs to be improved to automatically manage rate limits which is 200/hour or something. This would allow the program to be dockerized and hooked up like that.

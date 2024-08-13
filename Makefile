.PHONY: outdated


# Lists outdated packages
outdated:
	go list -u -m -json all | go-mod-outdated -update -direct 
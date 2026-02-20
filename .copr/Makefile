# COPR SCM build hook.
# COPR calls: make -f .copr/Makefile srpm outdir=<path>
# This Makefile vendors Go dependencies into the source tarball and produces an SRPM.

outdir ?= .

SPEC    := $(CURDIR)/ssh-tui.spec
NAME    := ssh-tui
VERSION := $(shell grep '^Version:' $(SPEC) | awk '{print $$2}')
TARBALL := $(NAME)-$(VERSION).tar.gz
WORKDIR := /tmp/copr-$(NAME)-$(VERSION)

.PHONY: srpm
srpm:
	# Install Go if not present (COPR SCM builder may not have it)
	command -v go >/dev/null 2>&1 || dnf install -y golang

	# Clean any leftover work directory
	rm -rf $(WORKDIR)
	mkdir -p $(WORKDIR)

	# Create source archive from the current git checkout
	git archive --prefix=$(NAME)-$(VERSION)/ HEAD \
		| gzip > $(WORKDIR)/$(TARBALL)

	# Extract, vendor Go dependencies, repack
	tar xf $(WORKDIR)/$(TARBALL) -C $(WORKDIR)/
	cd $(WORKDIR)/$(NAME)-$(VERSION) && go mod vendor
	tar czf $(WORKDIR)/$(TARBALL) -C $(WORKDIR) $(NAME)-$(VERSION)/

	# Build the SRPM
	rpmbuild -bs \
		--define "_sourcedir $(WORKDIR)" \
		--define "_srcrpmdir $(outdir)" \
		$(SPEC)

	# Clean up
	rm -rf $(WORKDIR)

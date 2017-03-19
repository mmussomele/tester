SHELL := /bin/bash
REPO = quilt

.PHONY: tests
tests:
	for suite in tests/* ; do \
		for f in $$suite/* ; do \
			if [ $${f: -3} == ".go" ] ; then \
				go build -o $${f%???} $$f ; \
			fi \
		done \
	done

docker-build:
	docker build -t ${REPO}/tester .

docker-push: docker-build
	docker push ${REPO}/tester

# Include all .mk files so you can have your own local configurations
include $(wildcard *.mk)

#
# Makefile
# @author Maksym Shkolnyi <maksymsh@wix.com>
#

.PHONY: bazel-purge bazel-clean gazelle-deps gazelle bazel-prep bazel-build bazel-cleanbuild bazel-build-from-scratch

bazel-purge:
	bazel clean --expunge

bazel-clean:
	bazel clean

gazelle-deps:
	bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=third_party.bzl%third_party_dependencies

gazelle:
	bazel run //:gazelle

bazel-prep: bazel-clean gazelle-deps gazelle

bazel-build:
	bazel build --verbose_failures  //:wtf

bazel-cleanbuild: bazel-prep bazel-build

bazel-build-from-scratch: bazel-purge gazelle-deps gazelle bazel-build
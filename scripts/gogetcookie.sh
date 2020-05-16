#!/bin/bash

touch ~/.gitcookies
chmod 0600 ~/.gitcookies

git config --global http.cookiefile ~/.gitcookies

tr , \\t <<\__END__ >>~/.gitcookies
go.googlesource.com,FALSE,/,TRUE,2147483647,o,git-luiz.hashicorp.com=1//0fXKRSfkRDWObCgYIARAAGA8SNwF-L9Iryt118o1nG9lyA7u2Br3k5615yndRjjqHRJUsBfoKgggZvePH1uMMn2sjh3zv8wXzLDs
go-review.googlesource.com,FALSE,/,TRUE,2147483647,o,git-luiz.hashicorp.com=1//0fXKRSfkRDWObCgYIARAAGA8SNwF-L9Iryt118o
__END__

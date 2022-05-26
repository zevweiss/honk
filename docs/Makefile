MANS = honk.1 honk.3 honk.5 honk.8 \
	hfcs.1 intro.1 vim.3 activitypub.7

HTMLS = $(foreach m,$(MANS),$m.html)

all: $(HTMLS)

$(HTMLS): %.html: %
	mandoc -T html -O style=mandoc.css,man=%N.%S.html $< > $@

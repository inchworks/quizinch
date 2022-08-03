## Step 5: Customise your system
Add files in `/srv/quizinch/site/` to customise your installation. You must restart the service for changes to take effect.

### Graphics
Files in `images/` replace the default brand and favicon images for QuizInch.

`brand.png` is the image shown on the site’s navbar. It should be 124px high. The width isn’t critical ; as a guide the default image is 558px wide.

[realfavicongenerator.net][1] was used to generate the default set of favicon files. If you want your own set, take care to generate all of these:
- android-chrome-192x192.png
- android-chrome-512x512.png
- apple-touch-icon.png
- apple-touch-icon-152x152-precomposed.png
- favicon.ico
- favicon-16x16.png
- favicon-32x32.png
- mstile-150x150.png
- safari-pinned-tab.svg

The following may be left unchanged (although realfavicongenerator.net will make them for you):
- browserconfig.xml
- site.webmanifest

You may also add add additional images you wish to include in customised templates to `/images`. They will be served as `static/images/*`. These files are intended to be unchanging; dynamic content should go in
`/srv/quizinch/media`.

### Configuration Parameters
The essential items are shown in docker-compose.yml. See [configuration.yml]({{ site.baseurl }}{% link configuration.yml.md %}) for the full set of options.

[1]:	https://realfavicongenerator.net
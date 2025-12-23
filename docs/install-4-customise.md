## Step 4: Customise your system
Add files in `/srv/quizinch/site/` to customise your installation. You must restart the service for changes to take effect.

### Graphics
Files in `images/` replace the default graphic and favicon images for QuizInch.

`big-logo.png` is the competition logo shown on the start, interval and end of quiz slides.

`strip.png` is the image shown on the left edge of each slide.

[realfavicongenerator.net][1] was used to generate the default set of favicon files. If you want your own set, take care to generate all of these:
 - favicon-96x96.png
 - favicon.svg
 - favicon.ico
 - apple-touch-icon.png
 - web-app-manifest-192x192.png
 - web-app-manifest-512x512.png

site.webmanifest may be left unchanged, although realfavicongenerator.net will make it for you.

### Configuration Parameters
The essential items are shown in docker-compose.yml. See [configuration.yml]({{ site.baseurl }}{% link configuration.yml.md %}) for the full set of options.

[1]:	https://realfavicongenerator.net
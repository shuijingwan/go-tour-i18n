// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"html"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCourseAdSPALifecycleInBrowser(t *testing.T) {
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}
	read := func(name string) []byte {
		data, err := fs.ReadFile(contentTour, name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	angular := read("tour/static/lib/angular.min.js")
	adScript := read("tour/static/go-dev/course-ad.js")
	directives := read("tour/static/js/directives.js")

	document := `<!doctype html><html ng-app="courseAdTest"><body><div ng-view></div>
<script type="text/ng-template" id="course-page"><div id="editor-container" class="course-body" data-page="{{page}}"><div class="go-dev-course-ad" data-go-dev-course-ad course-ad></div></div></script>
<script>` + string(angular) + `</script><script>` + string(adScript) + `</script><script>` + string(directives) + `</script><script>
(function() {
    'use strict';
    var requests = 0;
    var failPush = false;
    var uncaught = false;
    window.adsbygoogle = { push: function() { requests++; if (failPush) throw new Error('fake AdSense failure'); } };
    // file:// Chrome tests may deny sessionStorage; exercise the helper's
    // same-window fallback used when browser privacy settings do the same.
    window.__goDevCourseAdExperimentGroup = 'B';
    window.addEventListener('error', function() { uncaught = true; });
    angular.module('courseAdTest', ['ng', 'tour.directives']).config(['$routeProvider', '$locationProvider', function($routeProvider, $locationProvider) {
        $routeProvider.when('/:page', { templateUrl: 'course-page', controller: ['$scope', '$routeParams', function($scope, $routeParams) { $scope.page = $routeParams.page; }] });
        $locationProvider.hashPrefix('!');
    }]);

    function assert(condition, message) {
        if (!condition) throw new Error(message);
    }
    function waitFor(page, done) {
        var current = document.querySelector('#editor-container');
        if (current && current.getAttribute('data-page') === page) { done(current); return; }
        setTimeout(function() { waitFor(page, done); }, 10);
    }
    function navigate(path, done) {
        var injector = angular.element(document.documentElement).injector();
        injector.get('$rootScope').$apply(function() { injector.get('$location').path(path); });
        waitFor(path.slice(1), done);
    }
    function currentAdCount() {
        return document.querySelectorAll('[data-go-dev-course-ad] ins.adsbygoogle').length;
    }
    function assertExperimentB(container, label) {
        var ad = container.querySelector('ins.adsbygoogle');
        assert(container.getAttribute('data-go-dev-course-ad-group') === 'B', label + ': group changed');
        assert(container.classList.contains('go-dev-course-ad--max-468'), label + ': width strategy changed');
        assert(ad && ad.getAttribute('data-ad-slot') === '1260537939', label + ': slot changed');
        assert(ad.getAttribute('data-ad-format') === 'auto' && ad.getAttribute('data-full-width-responsive') === 'true', label + ': ad is no longer responsive');
    }

    setTimeout(function() {
        try {
            var injector = angular.element(document.documentElement).injector();
            injector.get('$rootScope').$apply(function() { injector.get('$location').path('/one'); });
            waitFor('one', function(first) {
                assert(document.querySelectorAll('[data-go-dev-course-ad]').length === 1, 'initial container count');
                assert(currentAdCount() === 1 && requests === 1, 'initial ad lifecycle');
                assertExperimentB(document.querySelector('[data-go-dev-course-ad]'), 'initial mount');
                navigate('/two', function() {
                    assert(!document.documentElement.contains(first), 'old view remains after navigation');
                    assert(first.querySelectorAll('ins.adsbygoogle').length === 0, 'old ad lifecycle did not end');
                    assert(document.querySelectorAll('[data-go-dev-course-ad]').length === 1 && currentAdCount() === 1 && requests === 2, 'second ad lifecycle');
                    assertExperimentB(document.querySelector('[data-go-dev-course-ad]'), 'second mount');
                    navigate('/three', function() {
                        navigate('/four', function() {
                            navigate('/five', function() {
                                assert(document.querySelectorAll('[data-go-dev-course-ad]').length === 1 && currentAdCount() === 1 && requests === 5, 'three SPA transitions accumulated ads');
                                assertExperimentB(document.querySelector('[data-go-dev-course-ad]'), 'later SPA mount');
                                failPush = true;
                                navigate('/failure', function() {
                                    assert(document.querySelector('.course-body') && currentAdCount() === 1, 'failed ad request broke the new view');
                                    failPush = false;
                                    navigate('/recovered', function() {
                                        assert(document.querySelector('.course-body') && currentAdCount() === 1 && !uncaught, 'navigation did not recover after failed ad request');
                                        var beforeRapid = requests;
                                        injector.get('$rootScope').$apply(function() {
                                            injector.get('$location').path('/rapid-one');
                                            injector.get('$location').path('/rapid-two');
                                        });
                                        waitFor('rapid-two', function() {
                                            assert(document.querySelectorAll('[data-go-dev-course-ad]').length === 1 && currentAdCount() === 1, 'rapid navigation left multiple ads');
                                            assert(requests === beforeRapid + 1, 'rapid navigation requested a stale view ad');
                                            document.body.setAttribute('data-course-ad-test', 'PASS');
                                        });
                                    });
                                });
                            });
                        });
                    });
                });
            });
        } catch (error) {
            document.body.setAttribute('data-course-ad-test', 'FAIL: ' + error.message);
        }
    }, 0);
}());
</script></body></html>`

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "course-ad-test.html")
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(chrome, "--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage", "--disable-breakpad", "--disable-crash-reporter", "--noerrdialogs", "--user-data-dir="+filepath.Join(tempDir, "chrome-profile"), "--run-all-compositor-stages-before-draw", "--virtual-time-budget=3000", "--dump-dom", "file://"+path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("google-chrome: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), `data-course-ad-test="PASS"`) {
		t.Fatalf("course ad browser test failed:\n%s", html.UnescapeString(string(output)))
	}
}

func TestCourseAdLayoutProtectionInBrowser(t *testing.T) {
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}
	read := func(name string) []byte {
		data, err := fs.ReadFile(contentTour, name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	document := `<!doctype html><html ng-app="courseAdLayoutTest"><head><style>
html, body { margin: 0; }
#editor-container { min-height: calc(100vh - 48px); }
#left-side, #left-side > .relative-content { min-height: calc(100vh - 48px); }
.site-footer { height: 48px; }
</style></head><body><div ng-view></div><footer class="site-footer">footer</footer>
<script type="text/ng-template" id="course-page"><div id="editor-container" data-page="{{page}}"><div class="relative-content"><div id="left-side"><div class="relative-content"><div class="go-dev-course-ad" data-go-dev-course-ad course-ad></div></div></div></div></div></script>
<script>` + string(read("tour/static/lib/angular.min.js")) + `</script><script>` + string(read("tour/static/go-dev/course-ad.js")) + `</script><script>` + string(read("tour/static/js/directives.js")) + `</script><script>
(function() {
    'use strict';
    var requests = 0;
    window.adsbygoogle = { push: function() { requests++; } };
    angular.module('courseAdLayoutTest', ['ng', 'tour.directives']).config(['$routeProvider', '$locationProvider', function($routeProvider, $locationProvider) {
        $routeProvider.when('/:page', { templateUrl: 'course-page', controller: ['$scope', '$routeParams', function($scope, $routeParams) { $scope.page = $routeParams.page; }] });
        $locationProvider.hashPrefix('!');
    }]);

    function assert(condition, message) {
        if (!condition) throw new Error(message);
    }
    function tick() {
        return new Promise(function(resolve) { setTimeout(resolve, 0); });
    }
    function waitFor(page) {
        return new Promise(function(resolve) {
            function check() {
                var editor = document.querySelector('#editor-container');
                if (editor && editor.getAttribute('data-page') === page) { resolve(editor); return; }
                setTimeout(check, 10);
            }
            check();
        });
    }
    function navigate(path) {
        angular.element(document.documentElement).injector().get('$rootScope').$apply(function() {
            angular.element(document.documentElement).injector().get('$location').path(path);
        });
        return waitFor(path.slice(1));
    }
    function nodes(editor) {
        var editorContent = editor.children[0];
        var leftSide = editorContent.children[0];
        return [editor, editorContent, leftSide, leftSide.children[0]];
    }
    function contaminate(targets) {
        targets.forEach(function(node) {
            node.style.setProperty('height', 'auto', 'important');
            node.style.setProperty('min-height', '0px', 'important');
            node.style.setProperty('width', '37px', 'important');
        });
    }
    function assertClean(targets, label) {
        targets.forEach(function(node) {
            assert(node.style.getPropertyValue('height') === '', label + ': height was not removed');
            assert(node.style.getPropertyValue('min-height') === '', label + ': min-height was not removed');
            assert(node.style.getPropertyValue('width') === '37px', label + ': unrelated style was removed');
        });
    }

    setTimeout(async function() {
        try {
            var first = await navigate('/one');
            var firstNodes = nodes(first);
            assert(requests === 1 && first.querySelectorAll('ins.adsbygoogle').length === 1, 'initial mount/push lifecycle');
            contaminate(firstNodes);
            await tick();
            assertClean(firstNodes, 'initial write');
            contaminate(firstNodes);
            await tick();
            assertClean(firstNodes, 'repeated write');

            var second = await navigate('/two');
            var secondNodes = nodes(second);
            assert(requests === 2 && first.querySelectorAll('ins.adsbygoogle').length === 0, 'first unmount lifecycle');
            firstNodes[0].style.setProperty('height', 'auto', 'important');
            contaminate(secondNodes);
            await tick();
            assert(firstNodes[0].style.getPropertyValue('height') === 'auto', 'old observer remained active');
            assertClean(secondNodes, 'second editor');

            var third = await navigate('/three');
            var thirdNodes = nodes(third);
            secondNodes[0].style.setProperty('height', 'auto', 'important');
            contaminate(thirdNodes);
            await tick();
            assert(secondNodes[0].style.getPropertyValue('height') === 'auto', 'second observer accumulated');
            assertClean(thirdNodes, 'third editor');
            assert(requests === 3 && document.querySelectorAll('[data-go-dev-course-ad] ins.adsbygoogle').length === 1, 'SPA mount/unmount lifecycle');

            var footer = document.querySelector('.site-footer');
            assert(third.offsetHeight >= window.innerHeight - 48, 'course body no longer occupies the viewport');
            assert(footer.offsetTop >= third.offsetTop + third.offsetHeight, 'footer entered the course body viewport');
            document.body.setAttribute('data-course-ad-layout-test', 'PASS');
        } catch (error) {
            document.body.setAttribute('data-course-ad-layout-test', 'FAIL: ' + error.message);
        }
    }, 0);
}());
</script></body></html>`

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "course-ad-layout-test.html")
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(chrome, "--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage", "--disable-breakpad", "--disable-crash-reporter", "--noerrdialogs", "--window-size=1280,800", "--user-data-dir="+filepath.Join(tempDir, "chrome-profile"), "--run-all-compositor-stages-before-draw", "--virtual-time-budget=3000", "--dump-dom", "file://"+path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("google-chrome: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), `data-course-ad-layout-test="PASS"`) {
		t.Fatalf("course ad layout browser test failed:\n%s", html.UnescapeString(string(output)))
	}
}

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

    setTimeout(function() {
        try {
            var injector = angular.element(document.documentElement).injector();
            injector.get('$rootScope').$apply(function() { injector.get('$location').path('/one'); });
            waitFor('one', function(first) {
                assert(document.querySelectorAll('[data-go-dev-course-ad]').length === 1, 'initial container count');
                assert(currentAdCount() === 1 && requests === 1, 'initial ad lifecycle');
                navigate('/two', function() {
                    assert(!document.documentElement.contains(first), 'old view remains after navigation');
                    assert(first.querySelectorAll('ins.adsbygoogle').length === 0, 'old ad lifecycle did not end');
                    assert(document.querySelectorAll('[data-go-dev-course-ad]').length === 1 && currentAdCount() === 1 && requests === 2, 'second ad lifecycle');
                    navigate('/three', function() {
                        navigate('/four', function() {
                            navigate('/five', function() {
                                assert(document.querySelectorAll('[data-go-dev-course-ad]').length === 1 && currentAdCount() === 1 && requests === 5, 'three SPA transitions accumulated ads');
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

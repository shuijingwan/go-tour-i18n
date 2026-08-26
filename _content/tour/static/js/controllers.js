/* Copyright 2012 The Go Authors.   All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */
'use strict';

/* Controllers */


angular.module('tour.controllers', []).

controller('ListCtrl', ['$scope', 'i18n', 'seo',
    function($scope, i18n, seo) {
        $scope.listHeading = i18n.l('tour.list_heading');
        seo.list();
    }
]).

// Navigation controller
controller('EditorCtrl', ['$scope', '$routeParams', '$location', 'toc', 'lessons', 'i18n', 'run', 'fmt', 'editor', 'analytics', 'storage', 'seo',
    function($scope, $routeParams, $location, toc, lessons, i18n, run, fmt, editor, analytics, storage, seo) {

        $scope.toc = toc;
        $scope.lessonId = $routeParams.lessonId;
        $scope.curPage = parseInt($routeParams.pageNumber);
        $scope.curFile = 0;
        $scope.job = null;
        $scope.editorUI = {
            syntax: i18n.l('editor.syntax'),
            imports: i18n.l('editor.imports'),
            run: i18n.l('editor.run'),
            kill: i18n.l('editor.kill'),
            format: i18n.l('editor.format'),
            reset: i18n.l('editor.reset')
        };

        $scope.nextPageClick = function(event) {
            event.preventDefault();
            $scope.nextPage();
        };
        $scope.prevPageClick = function(event) {
            event.preventDefault();
            $scope.prevPage();
        };
        $scope.nextPage = function() {
            $scope.gotoPage($scope.curPage + 1);
        };
        $scope.prevPage = function() {
            $scope.gotoPage($scope.curPage - 1);
        };
        $scope.gotoPage = function(page) {
            $scope.kill();
            var l = $routeParams.lessonId;
            if (page >= 1 && page <= lessons[$scope.lessonId].Pages.length) {
                $scope.curPage = page;
            } else {
                l = (page < 1) ? toc.prevLesson(l) : toc.nextLesson(l);
                if (l === '') { // If there's no previous or next
                    $location.path('/tour/list');
                    return;
                }
                page = (page < 1) ? lessons[l].Pages.length : 1;
            }
            $location.path('/tour/' + l + '/' + page);
            seo.page(l, page, lessons[l], lessons[l].Pages[page - 1]);
            $scope.openFile($scope.curFile);
            analytics.trackView();
        };
        $scope.openFile = function(file) {
            $scope.curFile = file;
            editor.paint();
        };

        function log(mode, text) {
            $('.output.active').html('<pre class="' + mode + '">' + text + '</pre>');
        }

        function clearOutput() {
            $('.output.active').html('');
        }

        function file() {
            return lessons[$scope.lessonId].Pages[$scope.curPage - 1].Files[$scope.curFile];
        }

        $scope.run = function() {
            log('info', i18n.l('execution.waiting'));
            var f = file();
            $scope.job = run(f.Content, $('.output.active > pre')[0], {
                path: f.Name
            }, function() {
                $scope.job = null;
                $scope.$apply();
            });
        };

        $scope.kill = function() {
            if ($scope.job !== null) $scope.job.Kill();
        };

        $scope.format = function() {
            log('info', i18n.l('execution.waiting'));
            fmt(file().Content, editor.imports).then(
                function(data) {
                    if (data.data.Error !== '') {
                        log('stderr', data.data.Error);
                        return;
                    }
                    clearOutput();
                    file().Content = data.data.Body;
                },
                function(error) {
                    log('stderr', error);
                });
        };

        $scope.reset = function() {
            file().Content = file().OrigContent;
        };

        $scope.gotoPage($scope.curPage);

        // Store changes on the current file to local storage.
        $scope.$watch(function() {
            var f = file();
            return f && f.Content;
        }, function(val) {
            if (val) storage.set(file().Hash, val);
        });
    }
]);

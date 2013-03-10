(function AppData() {
    'use strict';
    var viewModel = {
        titles: ko.mapping.fromJS([]),
        searchTerm: ko.observable(),
        reviews: ko.observable(),
        search: function() {
            var searchTerm = this.searchTerm();
            if (searchTerm === undefined || searchTerm === '') return;
            this.reviews({});
            $.ajax({
                url: 'api/1/search?q=' + searchTerm,
                context: this,
                success: this.searchCallback,
                error: function(request, textStatus, errorThrown) {
                    alert('search failed:', textStatus);
                }
            });
        },
        searchCallback: function(result) {
            ko.mapping.fromJS(result, this.titles);
            result.forEach(this.getReviews, this);
        },
        getReviews: function(elem) {
            var url = 'api/1/reviews/' + elem.id;
            $.ajax({
                url: url,
                context: this,
            })
            .success(function(data) {
                if (data.ratings.critics_score > 0) {
                    this.reviews()[elem.id] = data;
                    this.showReview(elem.id);
                }
            });
        },
        hasReview: function(id) {
            return this.reviews[id] != "undefined";
        },
        showReview: function(id) {
            if (this.hasReview(id)) {
                var data = this.reviews()[id];
                $("#" + id).html("Rotten Tomatoes score: " + data.ratings.critics_score);
            }
        }
    };

    ko.applyBindings(viewModel);
})();

/*(function () {

  defModules();
  bootstrap();

  function defModules() {
    define('jquery', [], function () { return root.jQuery; });
    define('ko', [], function () { return root.ko; });
    define('ko.mapping', [], function () { return root.ko.mapping; });
  }

  function bootstrap() {
    require(['bootstrap'], function (b) { b.boot(); });
  }
})();*/
// vim: set ts=4 sw=4 sts=4 et:

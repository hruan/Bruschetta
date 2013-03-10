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
                var score = this.reviews()[id].ratings.critics_score;
                var imgFresh = '<img class="rt-rating" src="/img/fresh.png" />';
                var imgRotten = '<img class="rt-rating" src="/img/rotten.png" />';
                var img = score >= 60 ? imgFresh : imgRotten;
                $("#" + id).html(img + score + '%');
            }
        },
        nfRating: function(rating) {
            var r = Math.floor(rating());
            var img = '<img class="star-rating" src="/img/star.png"/>';
            var imgPartial = '<img class="star-rating partial" src="/img/star-partial.png"/>';
            var imgEmpty = '<img class="star-rating partial" src="/img/star-empty.png"/>';
            var html = '';

            for (var i = 0; i < r; i++) {
                html += img;
            }

            if (r < rating()) {
                html += imgPartial;
            }

            for (var i = 0; i < 5 - Math.ceil(rating()); i++) {
                html += imgEmpty;
            }

            return html;
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

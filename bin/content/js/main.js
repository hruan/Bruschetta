(function AppData() {
    'use strict';
    var viewModel = {
        titles: ko.mapping.fromJS([]),
        searchTerm: ko.observable(),
        search: function() {
            var searchTerm = this.searchTerm();
            if (searchTerm === undefined || searchTerm === '') return;
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
            result.forEach(this.getReviews);
        },
        getReviews: function(elem) {
            var url = 'api/1/reviews/' + elem.id;
            $.getJSON(url, function(data) {
                $("#" + elem.id).html("Critics score: " + data.ratings.critics_score);
            })
            .error(function() {
                $("#" + elem.id).html("No reviews!");
            });
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

(function AppData() {
  var self = this;

  self.searchTerm = ko.observable();
  self.titles = ko.mapping.fromJS([]);
  self.search = function() {
    if (searchTerm === undefined || searchTerm === '') return;
    $.ajax({
      url: 'api/1/search?q=' + self.searchTerm(),
      success: function(result) {
        ko.mapping.fromJS(result, self.titles);
        result.forEach(function(elem) {
          var url = 'api/1/reviews/' + elem.year + '/' + hyphenify(elem.title);
          $.getJSON(url, function(data) {
            $("#" + elem.id).html("Critics score: " + data.ratings.critics_score);
          })
          .error(function() {
            $("#" + elem.id).html("No reviews!");
          });
        });
      },
      error: function(request, textStatus, errorThrown) {
        alert('search failed:', textStatus);
      }
    });
  }

  function hyphenify(str) {
    return str.replace(/[^\w- ]+/g, '')
      .replace(/[\s-]+/g, '-')
      .toLowerCase();
  }

  ko.applyBindings(this);
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
// vim: set ts=2 sw=2 sts=2 et:

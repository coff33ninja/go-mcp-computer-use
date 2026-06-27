using System;
using System.IO;
using System.Runtime.InteropServices.WindowsRuntime;
using Windows.Graphics.Imaging;
using Windows.Media.Ocr;
using Windows.Storage;
using Windows.Storage.Streams;

class OcrHelper {
  static int Main(string[] args) {
    if (args.Length < 1) { Console.Error.WriteLine("Usage: ocrhelper.exe <image.png> [language]"); return 1; }
    string path = args[0];
    string lang = args.Length > 1 ? args[1] : null;
    try {
      var file = (IStorageFile)StorageFile.GetFileFromPathAsync(path).AsTask().GetAwaiter().GetResult();
      using (var stream = file.OpenReadAsync().AsTask().GetAwaiter().GetResult()) {
        var decoder = BitmapDecoder.CreateAsync(stream).AsTask().GetAwaiter().GetResult();
        var sb = decoder.GetSoftwareBitmapAsync().AsTask().GetAwaiter().GetResult();
        OcrEngine engine;
        if (!string.IsNullOrEmpty(lang))
          engine = OcrEngine.TryCreateFromLanguage(new Windows.Globalization.Language(lang));
        else
          engine = OcrEngine.TryCreateFromUserProfileLanguages();
        if (engine == null) { Console.WriteLine("{\"text\":\"\",\"lines\":[],\"words\":[]}"); return 0; }
        var result = engine.RecognizeAsync(sb).AsTask().GetAwaiter().GetResult();
        if (result == null) { Console.WriteLine("{\"text\":\"\",\"lines\":[],\"words\":[]}"); return 0; }
        Console.Write("{\"text\":");
        Console.Write(JsonEncode(result.Text));
        Console.Write(",\"lines\":[");
        bool first = true;
        foreach (var line in result.Lines) {
          if (!first) Console.Write(",");
          first = false;
          Console.Write("{\"text\":" + JsonEncode(line.Text));
          Console.Write(",\"x\":" + line.BoundingRect.X + ",\"y\":" + line.BoundingRect.Y);
          Console.Write(",\"w\":" + line.BoundingRect.Width + ",\"h\":" + line.BoundingRect.Height + "}");
        }
        Console.Write("],\"words\":[");
        first = true;
        foreach (var line in result.Lines) {
          foreach (var word in line.Words) {
            if (!first) Console.Write(",");
            first = false;
            Console.Write("{\"text\":" + JsonEncode(word.Text));
            Console.Write(",\"x\":" + word.BoundingRect.X + ",\"y\":" + word.BoundingRect.Y);
            Console.Write(",\"w\":" + word.BoundingRect.Width + ",\"h\":" + word.BoundingRect.Height + "}");
          }
        }
        Console.WriteLine("]}");
        return 0;
      }
    } catch (Exception ex) {
      Console.Error.WriteLine("OCR error: " + ex.Message);
      return 1;
    }
  }

  static string JsonEncode(string s) {
    if (s == null) return "\"\"";
    return "\"" + s.Replace("\\", "\\\\").Replace("\"", "\\\"").Replace("\r", "\\r").Replace("\n", "\\n").Replace("\t", "\\t") + "\"";
  }
}

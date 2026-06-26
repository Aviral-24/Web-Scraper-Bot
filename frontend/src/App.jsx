import React, { useState } from 'react';

function App() {
  const [formData, setFormData] = useState({
    targetUrl: 'https://quotes.toscrape.com/',
    containerSelector: '.quote',
    titleSelector: '.text',
    priceSelector: '.author'
  });

  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleChange = (e) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleScrape = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setData(null);

    try {
      //Live deploy karte waqt localhost:8080 ko Render URL se replace karein
      const response = await fetch('http://localhost:8080/api/scrape', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });

      const result = await response.json();
      
      if (!response.ok) throw new Error(result.error || 'Scraping failed');
      
      setData(result);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleDownloadCSV = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/scrape', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...formData, export: 'csv' }),
      });

      if (!response.ok) throw new Error('CSV Export failed');

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'scraped_data.csv';
      document.body.appendChild(a);
      a.click();
      a.remove();
    } catch (err) {
      alert("Error downloading CSV: " + err.message);
    }
  };

  return (
    <div className="min-h-screen p-8 text-gray-800 bg-gray-50">
      <div className="max-w-6xl mx-auto space-y-8">
        
        <div className="text-center">
          <h1 className="text-4xl font-extrabold text-indigo-600 tracking-tight">Web Scraper Bot</h1>
          <p className="mt-2 text-gray-500">High-Speed Concurrent Web Scraper Engine</p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          
          <div className="lg:col-span-1 bg-white p-6 rounded-2xl shadow-sm border border-gray-200">
            <h2 className="text-xl font-bold mb-4 text-gray-700">Target Configuration</h2>
            <form onSubmit={handleScrape} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">Target URL</label>
                <input type="url" name="targetUrl" value={formData.targetUrl} onChange={handleChange} required
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:outline-none" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">Container Selector</label>
                <input type="text" name="containerSelector" value={formData.containerSelector} onChange={handleChange} required
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:outline-none" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">Title Selector</label>
                <input type="text" name="titleSelector" value={formData.titleSelector} onChange={handleChange} required
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:outline-none" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">Price/Data Selector</label>
                <input type="text" name="priceSelector" value={formData.priceSelector} onChange={handleChange} required
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:outline-none" />
              </div>
              <button type="submit" disabled={loading}
                className="w-full mt-4 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold py-3 px-4 rounded-lg transition duration-200 shadow-md">
                {loading ? 'Scraping Data...' : 'Start Scraping'}
              </button>
            </form>
          </div>

          <div className="lg:col-span-2 bg-white p-6 rounded-2xl shadow-sm border border-gray-200 flex flex-col">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-bold text-gray-700">Live Results</h2>
              {data && data.data && (
                <button onClick={handleDownloadCSV}
                  className="bg-green-500 hover:bg-green-600 text-white px-4 py-2 rounded-lg font-medium transition duration-200 shadow-sm flex items-center gap-2">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path></svg>
                  Export CSV
                </button>
              )}
            </div>

            {loading && (
              <div className="flex-1 flex flex-col items-center justify-center text-gray-500 py-12">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mb-4"></div>
                <p>Deploying concurrent bots...</p>
              </div>
            )}

            {error && (
              <div className="bg-red-50 text-red-600 p-4 rounded-lg border border-red-200">
                <strong>Error:</strong> {error}
              </div>
            )}

            {!loading && !error && !data && (
              <div className="flex-1 flex items-center justify-center text-gray-400 py-12 border-2 border-dashed border-gray-200 rounded-xl">
                Ready to extract data. Enter configuration to begin.
              </div>
            )}

            {!loading && data && data.data && (
              <div className="overflow-auto flex-1 max-h-[500px] border border-gray-200 rounded-xl">
                <table className="w-full text-left border-collapse">
                  <thead className="bg-gray-50 sticky top-0 border-b border-gray-200">
                    <tr>
                      <th className="py-3 px-4 text-sm font-semibold text-gray-600">Extracted Title</th>
                      <th className="py-3 px-4 text-sm font-semibold text-gray-600">Extracted Data</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {data.data.map((item, index) => (
                      <tr key={index} className="hover:bg-indigo-50 transition duration-150">
                        <td className="py-3 px-4 text-sm text-gray-800">{item.title}</td>
                        <td className="py-3 px-4 text-sm font-medium text-indigo-600">{item.price}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            
            {data && <div className="mt-4 text-sm text-gray-500 text-right">Total Items Scraped: {data.total}</div>}
          </div>

        </div>
      </div>
    </div>
  );
}

export default App;
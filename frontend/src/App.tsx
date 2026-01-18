import { HashRouter as Router, Routes, Route } from "react-router-dom";
import Layout from "./components/Layout";
import Library from "./pages/Library";
import MissingUpdate from "./pages/MissingUpdate";
import MissingDLC from "./pages/MissingDLC";
import Organize from "./pages/Organize";
import Issues from "./pages/Issues";
import "./App.css";
import { ThemeProvider } from "@/components/theme-provider";

function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <Router>
        <Layout>
          <Routes>
            <Route path="/" element={<Library />} />
            <Route path="/missing-update" element={<MissingUpdate />} />
            <Route path="/missing-dlc" element={<MissingDLC />} />
            <Route path="/organize" element={<Organize />} />
            <Route path="/issues" element={<Issues />} />
          </Routes>
        </Layout>
      </Router>
    </ThemeProvider>
  );
}

export default App;

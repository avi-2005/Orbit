import { Component } from "react";

export default class GlobeErrorBoundary extends Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidCatch(error) {
    console.error("Globe render error:", error);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="w-full h-full flex items-center justify-center">
          <div className="text-center text-white/50 mono text-sm px-6">
            <p>3D view failed to load.</p>
            <button
              onClick={() => window.location.reload()}
              className="mt-3 px-3 py-1.5 border border-white/20 rounded hover:bg-white/10 transition-colors"
            >
              Reload
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}

import React, { useState } from 'react';
import axios from 'axios';

function FileUpload({ onUpload }) {
  const [file, setFile] = useState(null);

  const uploadFile = () => {
    const formData = new FormData();
    formData.append('file', file);
    axios.post('/api/chat/upload', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
      .then(({ data }) => onUpload(`/uploads/${data.filename}`));
  };

  return (
    <div>
      <input type="file" onChange={e => setFile(e.target.files[0])} />
      <button onClick={uploadFile}>Upload</button>
    </div>
  );
}

export default FileUpload;